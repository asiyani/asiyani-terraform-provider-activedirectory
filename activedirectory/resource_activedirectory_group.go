package activedirectory

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	groupScopeDomainLocal = "domain_local"
	groupScopeGlobal      = "global"
	groupScopeUniversal   = "universal"
	groupTypeSecurity     = "security"
	groupTypeDistribution = "distribution"
)

func resourceActivedirectoryGroup() *schema.Resource {
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
					if len(v) > 20 {
						errs = append(errs, fmt.Errorf("sAMAccountName attribute is limited to MAX 20 characters, got value:%s count:%d", v, len(v)))
					}
					return
				},
			},
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The group scope, allowed values are 'domain_local','global' and 'universal'",
				Default:     "global",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != groupScopeDomainLocal && v != groupScopeGlobal && v != groupScopeUniversal {
						errs = append(errs, fmt.Errorf("invalid value provided for 'scope' argument, allowed values are 'domain_local','global', and 'universal', got value: %s", v))
					}
					return
				},
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The group type, allowed values are 'security' and 'distribution'",
				Default:     "security",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != groupTypeSecurity && v != groupTypeDistribution {
						errs = append(errs, fmt.Errorf("invalid value provided for 'type' argument, allowed values are 'security' and 'distribution', got value: %s", v))

					}
					return
				},
			},
			"members": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The member attribute of the AD object. contains object's DN.",
				Set:         lowercaseHashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the AD object",
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
			"attributes": {
				Type:         schema.TypeString,
				Description:  `The list of other attributes of object, represented in json as map with 'attribute name' as key and values as array of string ie '{attribute_name = ["value1","value2"]}'`,
				Optional:     true,
				ValidateFunc: validateAttributesJSON,
				StateFunc:    normalizeAttributesJSON,
				Default:      "{}",
			},
		},
		Create: resourceCreateGroup,
		Read:   resourceReadGroup,
		Update: resourceUpdateGroup,
		Delete: resourceDeleteObject,
		// Exists: resourceExistsObject,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceCreateGroup(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceCreateGroup: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	ou := d.Get("base_ou_dn").(string)

	if err := validateDNString(c, ou); err != nil {
		return fmt.Errorf("resourceCreateGroup: base_ou_dn is not valid err: %w", err)
	}

	addReq := groupSchemaToAddRequest(d)
	c.logger.Debug("resourceCreateGroup: ldap add request", "addReq", addReq)

	guid, err := addObject(c.conn, addReq)
	if err != nil {
		return fmt.Errorf("resourceCreateGroup: unable to create user err: %w", err)
	}
	c.logger.Info("resourceCreateGroup: group added to active directory", "guid", guid)
	d.SetId(guid)
	return resourceReadGroup(d, meta)
}

func resourceReadGroup(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceReadGroup: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	id, err := encodeGUID(d.Id())
	if err != nil {
		return fmt.Errorf("resourceReadGroup: unable to encode GUID:%v err:%w", d.Id(), err)
	}
	entry, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			c.logger.Error("resourceReadGroup: object not found", "GUID", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceReadGroup: unable to search group with ID  GUID:%v err:%w", d.Id(), err)
	}
	c.logger.Info("resourceReadGroup: group object found", "dn", entry.DN)

	if err := updateObjectSchema(resourceActivedirectoryGroup().Schema, entry, d); err != nil {
		return err
	}
	return nil
}

func resourceUpdateGroup(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceUpdateGroup: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	// check if DN is changed
	if d.HasChanges("name", "base_ou_dn") {
		oldName, newName := d.GetChange("name")
		oldOU, newOU := d.GetChange("base_ou_dn")
		c.logger.Debug("resourceUpdateGroup: Name Changes", "old", oldName, "new", newName)
		c.logger.Debug("resourceUpdateGroup: OU Changes", "old", oldOU, "new", newOU)

		if err := validateDNString(c, newOU.(string)); err != nil {
			return fmt.Errorf("resourceUpdateGroup: new base_ou_dn is not valid err: %w", err)
		}

		req := &ldap.ModifyDNRequest{
			DN:           "cn=" + oldName.(string) + "," + oldOU.(string),
			NewRDN:       "cn=" + newName.(string),
			DeleteOldRDN: true,
			NewSuperior:  newOU.(string),
		}
		if err = c.conn.ModifyDN(req); err != nil {
			return fmt.Errorf("resourceUpdateGroup: unable to update dn of LDAP object: ModifyDNRequest:%v err:%w", req, err)
		}
		c.logger.Info("resourceUpdateGroup: group DN modified", "NewRDN", req.NewRDN, "newOU", req.NewSuperior)

	}

	modReq := &ldap.ModifyRequest{DN: "cn=" + d.Get("name").(string) + "," + d.Get("base_ou_dn").(string)}

	if d.HasChange("sam_account_name") {
		modReq.Replace("sAMAccountName", []string{d.Get("sam_account_name").(string)})
		c.logger.Debug("resourceUpdateGroup: updating 'sAMAccountName'", "new", d.Get("sam_account_name").(string))

	}

	if d.HasChange("description") {
		if d.Get("description").(string) == "" {
			modReq.Replace("description", []string{})
		} else {
			modReq.Replace("description", []string{d.Get("description").(string)})
		}
		c.logger.Debug("resourceUpdateGroup: updating 'description'", "new", d.Get("description").(string))

	}

	// check if groupType is changed
	if d.HasChanges("scope", "type") {
		gTypeV := getGroupTypeValue(d.Get("scope").(string), d.Get("type").(string))
		modReq.Replace("groupType", []string{gTypeV})
		c.logger.Debug("resourceUpdateGroup: updating 'groupType'", "new", gTypeV)
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
			c.logger.Debug("resourceUpdateGroup: Replacing attribute", "name", name, "value", values)
		}
	}

	if len(modReq.Changes) > 0 {
		if err = c.conn.Modify(modReq); err != nil {
			return fmt.Errorf("resourceUpdateGroup: unable to update some attributes of LDAP object: ModifyRequest:%#v err:%w", modReq, err)
		}
		c.logger.Info("resourceUpdateGroup: modified", "dn", modReq.DN)
	}
	return resourceReadGroup(d, meta)
}

func groupSchemaToAddRequest(d *schema.ResourceData) *ldap.AddRequest {
	var addReq ldap.AddRequest

	name := d.Get("name").(string)
	attributes := d.Get("attributes").(string)

	addReq.DN = "cn=" + name + "," + d.Get("base_ou_dn").(string)

	// add attributes
	attrMap := map[string][]string{}
	_ = json.Unmarshal([]byte(attributes), &attrMap)
	for name, values := range attrMap {
		addReq.Attribute(name, values)
	}

	gTypeV := getGroupTypeValue(d.Get("scope").(string), d.Get("type").(string))
	addReq.Attribute("groupType", []string{gTypeV})

	addReq.Attribute("sAMAccountName", []string{d.Get("sam_account_name").(string)})
	addReq.Attribute("objectClass", []string{"group"})
	addReq.Attribute("name", []string{name})
	addReq.Attribute("cn", []string{name})
	if d.Get("description").(string) != "" {
		addReq.Attribute("description", []string{d.Get("description").(string)})
	}

	return &addReq
}
