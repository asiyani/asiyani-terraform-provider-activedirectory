package activedirectory

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceActivedirectoryOU() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"guid": {
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
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the AD object",
			},
			"ou": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name property of the OU",
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
		Create: resourceCreateOU,
		Read:   resourceReadOU,
		Update: resourceUpdateOU,
		Delete: resourceDeleteObject,
		// Exists: resourceExistsObject,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceCreateOU(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceCreateOU: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	ou := d.Get("base_ou_dn").(string)

	if err := validateDNString(c, ou); err != nil {
		return fmt.Errorf("resourceCreateOU: base_ou_dn is not valid err: %w", err)
	}

	addReq := ouSchemaToAddRequest(d)

	c.logger.Debug("resourceCreateOU: ldap add request", "addReq", addReq)

	guid, err := addObject(c.conn, addReq)
	if err != nil {
		return fmt.Errorf("resourceCreateOU: unable to create user err: %w", err)
	}
	c.logger.Info("resourceCreateOU: ou added to active directory", "guid", guid)
	d.SetId(guid)
	return resourceReadOU(d, meta)
}

func resourceReadOU(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceReadOU: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	id, err := encodeGUID(d.Id())
	if err != nil {
		return fmt.Errorf("resourceReadOU: unable to encode GUID:%v err:%w", d.Id(), err)
	}
	e, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			c.logger.Error("resourceReadOU: object not found", "GUID", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceReadOU: unable to search ou with ID  GUID:%v err:%w", d.Id(), err)
	}
	c.logger.Info("resourceReadOU: ou object found", "dn", e.DN)

	if err := updateObjectSchema(resourceActivedirectoryOU().Schema, e, d); err != nil {
		return err
	}
	return nil
}

func resourceUpdateOU(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceUpdateOU: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	// check if DN is changed
	if d.HasChanges("name", "base_ou_dn") {
		oldName, newName := d.GetChange("name")
		oldOU, newOU := d.GetChange("base_ou_dn")
		c.logger.Debug("resourceUpdateOU: Name Changes", "old", oldName, "new", newName)
		c.logger.Debug("resourceUpdateOU: OU Changes", "old", oldOU, "new", newOU)

		if err := validateDNString(c, newOU.(string)); err != nil {
			return fmt.Errorf("resourceUpdateOU: new base_ou_dn is not valid err: %w", err)
		}

		req := &ldap.ModifyDNRequest{
			DN:           "ou=" + oldName.(string) + "," + oldOU.(string),
			NewRDN:       "ou=" + newName.(string),
			DeleteOldRDN: true,
			NewSuperior:  newOU.(string),
		}
		if err = c.conn.ModifyDN(req); err != nil {
			return fmt.Errorf("resourceUpdateOU: unable to update dn of LDAP object: ModifyDNRequest:%v err:%w", req, err)
		}
		c.logger.Debug("resourceUpdateOU: OU DN modified", "NewRDN", req.NewRDN, "newOU", req.NewSuperior)
	}

	modReq := &ldap.ModifyRequest{DN: "ou=" + d.Get("name").(string) + "," + d.Get("base_ou_dn").(string)}

	if d.HasChange("description") {
		if d.Get("description").(string) == "" {
			modReq.Replace("description", []string{})
		} else {
			modReq.Replace("description", []string{d.Get("description").(string)})
		}
		c.logger.Debug("resourceUpdateOU: updating 'description'", "new", d.Get("description").(string))
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
			c.logger.Debug("resourceUpdateOU: Replacing attribute", "name", name, "value", values)
		}
	}

	if len(modReq.Changes) > 0 {
		if err = c.conn.Modify(modReq); err != nil {
			return fmt.Errorf("resourceUpdateOU: unable to update some attributes of LDAP object:%s err:%w", modReq.DN, err)
		}
		c.logger.Info("resourceUpdateOU: modified", "dn", modReq.DN)
	}
	return resourceReadOU(d, meta)
}

func ouSchemaToAddRequest(d *schema.ResourceData) *ldap.AddRequest {
	var addReq ldap.AddRequest

	name := d.Get("name").(string)
	attributes := d.Get("attributes").(string)

	addReq.DN = "ou=" + name + "," + d.Get("base_ou_dn").(string)

	// add attributes
	attrMap := map[string][]string{}
	_ = json.Unmarshal([]byte(attributes), &attrMap)
	for name, values := range attrMap {
		addReq.Attribute(name, values)
	}

	addReq.Attribute("objectClass", []string{"organizationalUnit"})
	addReq.Attribute("name", []string{name})
	addReq.Attribute("ou", []string{name})
	if d.Get("description").(string) != "" {
		addReq.Attribute("description", []string{d.Get("description").(string)})
	}

	return &addReq
}
