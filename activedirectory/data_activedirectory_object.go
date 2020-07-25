package activedirectory

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataActivedirectoryObject() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"guid": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dn": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The distinguished name (dn) of the object",
				DiffSuppressFunc: ignoreCaseDiffSuppressor,
			},
			"sam_account_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The sAMAccountName attribute is a logon name used to support clients and servers from previous version of Windows. The name must be 20 or fewer characters.",
			},
			"sid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the Object",
			},
			"base_ou_dn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full path of the Organizational Unit (OU) or container where the object is created",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A description of the AD object",
			},
			"cn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Common-Name property of the object",
			},
			"members": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The member attribute of the AD object. contains object's DN.",
				Set:         lowercaseHashString,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"member_of": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The memberOf attribute of the AD object. contains object's DN.",
				Set:         lowercaseHashString,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"user_principal_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The userPrincipalName for user object. should be in format `someone@domain.com`.",
			},
		},

		Read: dataReadObject,
	}
}

func dataReadObject(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("dataReadObject: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	var e *ldap.Entry
	objectGuid := d.Get("guid").(string)
	dn := d.Get("dn").(string)

	if objectGuid == "" && dn == "" {
		return fmt.Errorf("specify either 'guid' or 'dn' to search object.")
	}

	if objectGuid != "" {
		id, err := encodeGUID(objectGuid)
		if err != nil {
			return fmt.Errorf("dataReadObject: unable to encode GUID:%v err:%w", d.Id(), err)
		}
		e, err = getObjectByID(c, id)
		if err != nil {
			return fmt.Errorf("dataReadObject: unable to search object with GUID: %v err: %w", d.Id(), err)
		}
	} else {
		if err := validateDNString(c, dn); err != nil {
			return fmt.Errorf("resourceCreateUser: base_ou_dn is not valid err: %w", err)
		}
		e, err = getObjectByDN(c.conn, dn)
		if err != nil {
			return fmt.Errorf("dataReadObject: unable to search object with dn: %v err: %w", dn, err)
		}
	}

	c.logger.Debug("dataReadObject: object object found", "dn", e.DN)

	if err := updateObjectSchema(dataActivedirectoryObject().Schema, e, d); err != nil {
		return err
	}

	rawGuid := e.GetRawAttributeValue("objectGUID")
	guid, err := decodeGUID(rawGuid)
	if err != nil {
		return fmt.Errorf("dataReadObject: unable to convert raw GUID to string rawGUID:%x err:%w", rawGuid, err)
	}
	d.SetId(guid)

	return nil
}
