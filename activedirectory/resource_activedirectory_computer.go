package activedirectory

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceActivedirectoryComputer() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"guid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"sid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "The enabled status of Object, default is true",
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The name of the Object",
				DiffSuppressFunc: ignoreCaseDiffSuppressor,
			},
			"base_ou_dn": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The full path of the Organizational Unit (OU) or container where the object is created",
				DiffSuppressFunc: ignoreCaseDiffSuppressor,
			},
			"sam_account_name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The sAMAccountName attribute is a logon name used to support clients and servers from previous version of Windows. The name must be 20 or fewer characters.",
				DiffSuppressFunc: ignoreCaseDiffSuppressor,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if !strings.HasSuffix(v, "$") {
						errs = append(errs, fmt.Errorf("sAMAccountName attribute of a computer object should have trailing dollar sign ('$'), got: %s", v))
					}
					if len(v) > 20 {
						errs = append(errs, fmt.Errorf("sAMAccountName attribute is limited to MAX 20 characters, got value:%s count:%d", v, len(v)))
					}
					return
				},
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the AD object",
			},
			"user_account_control": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The userAccountControl of the object in decimal string value",
			},
			"cn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Common-Name property of the object",
			},
			"dn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The distinguished name (dn) of the object",
			},
			"member_of": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The memberOf attribute of the AD object. contains object's DN.",
				Set:         lowercaseHashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"attributes": {
				Type:         schema.TypeString,
				Description:  `The list of other attributes of object, represented in json as map with 'attribute name' as key and values as array of string ie '{attribute_name = ["value1","value2"]}'`,
				Optional:     true,
				ValidateFunc: validateAttributesJSON,
				StateFunc:    normalizeAttributesJSON,
				Default:      "{}",
			},
		},
		Create: resourceCreateComputer,
		Read:   resourceReadComputer,
		Update: resourceUpdateComputer,
		Delete: resourceDeleteObject,
		// Exists: resourceExistsObject,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceCreateComputer(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceCreateComputer: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	ou := d.Get("base_ou_dn").(string)

	if err := validateDNString(c, ou); err != nil {
		return fmt.Errorf("resourceCreateComputer: base_ou_dn is not valid err: %w", err)
	}

	addReq, err := computerSchemaToAddRequest(d)
	if err != nil {
		return fmt.Errorf("resourceCreateComputer: unable to convert schema to addrequest err:%w", err)
	}
	c.logger.Debug("resourceCreateComputer: ldap add request", "addReq", addReq)

	guid, err := addObject(c.conn, addReq)
	if err != nil {
		return fmt.Errorf("resourceCreateComputer: unable to create user err: %w", err)
	}
	c.logger.Info("resourceCreateComputer: computer added to active directory", "guid", guid)
	d.SetId(guid)
	return resourceReadComputer(d, meta)
}

func resourceReadComputer(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceReadComputer: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	id, err := encodeGUID(d.Id())
	if err != nil {
		return fmt.Errorf("resourceReadComputer: unable to encode GUID:%v err:%w", d.Id(), err)
	}
	e, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			c.logger.Error("resourceReadComputer: object not found", "GUID", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceReadComputer: unable to search computer with ID  GUID:%v err:%w", d.Id(), err)
	}
	c.logger.Info("resourceReadComputer: computer object found", "dn", e.DN)

	if err := updateObjectSchema(resourceActivedirectoryComputer().Schema, e, d); err != nil {
		return err
	}
	return nil
}

func resourceUpdateComputer(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceUpdateComputer: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	// check if DN is changed
	if d.HasChanges("name", "base_ou_dn") {
		oldName, newName := d.GetChange("name")
		oldOU, newOU := d.GetChange("base_ou_dn")
		c.logger.Debug("resourceUpdateComputer: Name Changes", "old", oldName, "new", newName)
		c.logger.Debug("resourceUpdateComputer: OU Changes", "old", oldOU, "new", newOU)

		if err := validateDNString(c, newOU.(string)); err != nil {
			return fmt.Errorf("resourceUpdateComputer: new base_ou_dn is not valid err: %w", err)
		}

		req := &ldap.ModifyDNRequest{
			DN:           "cn=" + oldName.(string) + "," + oldOU.(string),
			NewRDN:       "cn=" + newName.(string),
			DeleteOldRDN: true,
			NewSuperior:  newOU.(string),
		}
		if err = c.conn.ModifyDN(req); err != nil {
			return fmt.Errorf("resourceUpdateComputer: unable to update dn of LDAP object: ModifyDNRequest:%v err:%w", req, err)
		}
		c.logger.Info("resourceUpdateComputer: computer DN modified", "NewRDN", req.NewRDN, "newOU", req.NewSuperior)
	}

	modReq := &ldap.ModifyRequest{DN: "cn=" + d.Get("name").(string) + "," + d.Get("base_ou_dn").(string)}

	// check for other arguments and attributes changes
	if d.HasChange("enabled") {
		enabled := d.Get("enabled").(bool)
		uac := d.Get("user_account_control").(string)
		if enabled {
			uac, err = unsetaccountDisabledFlag(uac)
			if err != nil {
				return fmt.Errorf("resourceUpdateComputer: unable to unset account Disabled Flag for  userAccountControl value:%v ,err:%w", uac, err)
			}
		} else {
			uac, err = setaccountDisabledFlag(uac)
			if err != nil {
				return fmt.Errorf("resourceUpdateComputer: unable to set account Disabled Flag for  userAccountControl value:%v ,err:%w", uac, err)
			}
		}
		modReq.Replace("userAccountControl", []string{uac})
		c.logger.Debug("resourceUpdateComputer: updating 'userAccountControl'", "new", uac)
	}

	if d.HasChange("sam_account_name") {
		modReq.Replace("sAMAccountName", []string{d.Get("sam_account_name").(string)})
		c.logger.Debug("resourceUpdateComputer: updating 'sam_account_name'", "new", d.Get("sam_account_name").(string))
	}

	if d.HasChange("description") {
		if d.Get("description").(string) == "" {
			modReq.Replace("description", []string{})
		} else {
			modReq.Replace("description", []string{d.Get("description").(string)})
		}
		c.logger.Debug("resourceUpdateComputer: updating 'description'", "new", d.Get("description").(string))
	}

	if d.HasChange("attributes") {
		oldAttrMap := map[string][]string{}
		newAttrMap := map[string][]string{}

		oldAttr, newAttr := d.GetChange("attributes")
		_ = json.Unmarshal([]byte(oldAttr.(string)), &oldAttrMap)
		_ = json.Unmarshal([]byte(newAttr.(string)), &newAttrMap)

		replaced := getModifiedAttributes(oldAttrMap, newAttrMap)
		for name, values := range replaced {
			modReq.Replace(name, values)
			c.logger.Debug("resourceUpdateComputer: Replacing 'attribute'", "name", name, "new_value", values)
		}
	}

	if len(modReq.Changes) > 0 {
		if err = c.conn.Modify(modReq); err != nil {
			return fmt.Errorf("resourceUpdateComputer: unable to update some attributes of LDAP object: ModifyRequest:%#v err:%w", modReq, err)
		}
		c.logger.Info("resourceUpdateComputer: modified", "dn", modReq.DN)
	}
	return resourceReadComputer(d, meta)
}

func computerSchemaToAddRequest(d *schema.ResourceData) (*ldap.AddRequest, error) {
	var addReq ldap.AddRequest
	enabled := d.Get("enabled").(bool)

	name := d.Get("name").(string)
	attributes := d.Get("attributes").(string)
	uac := d.Get("user_account_control").(string)

	// set default value for computer userAccountControl to WORKSTATION_TRUST_ACCOUNT
	if uac == "" {
		uac = "4096"
	}

	addReq.DN = "cn=" + name + "," + d.Get("base_ou_dn").(string)

	// add attributes
	attrMap := map[string][]string{}
	_ = json.Unmarshal([]byte(attributes), &attrMap)
	for name, values := range attrMap {
		addReq.Attribute(name, values)
	}

	// verify userAccountControl value matches status provided
	uacStatus, err := isObjectEnabled(uac)
	if err != nil {
		return nil, fmt.Errorf("unable to verify status of given userAccountControl flag value:%v ,err:%w", uac, err)
	}
	if enabled != uacStatus && enabled {
		uac, err = unsetaccountDisabledFlag(uac)
		if err != nil {
			return nil, fmt.Errorf("unable to setaccountDisabledFlag for  userAccountControl value:%v ,err:%w", uac, err)
		}
	}
	if enabled != uacStatus && !enabled {
		uac, err = setaccountDisabledFlag(uac)
		if err != nil {
			return nil, fmt.Errorf("unable to unsetaccountDisabledFlag for userAccountControl value:%v ,err:%w", uac, err)
		}
	}

	addReq.Attribute("sAMAccountName", []string{d.Get("sam_account_name").(string)})
	addReq.Attribute("userAccountControl", []string{uac})
	addReq.Attribute("objectClass", []string{"computer"})
	addReq.Attribute("name", []string{name})
	addReq.Attribute("cn", []string{name})
	if d.Get("description").(string) != "" {
		addReq.Attribute("description", []string{d.Get("description").(string)})
	}

	return &addReq, nil
}
