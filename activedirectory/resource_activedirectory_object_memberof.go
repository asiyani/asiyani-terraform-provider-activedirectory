package activedirectory

import (
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceActivedirectoryObjectMemberOf() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"object_dn": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The AD object's dn to add in groups, should be of computer or user dn",
				DiffSuppressFunc: ignoreCaseDiffSuppressor,
			},
			"member_of": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of group's dns to add AD Object.",
				Set:         lowercaseHashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
						v := val.(string)
						if _, err := ldap.ParseDN(v); err != nil {
							errs = append(errs, fmt.Errorf("member_of entry should be valid DN, got value:%s err:%v", v, err))
						}
						return
					},
				},
			},
		},
		Create: resourceCreateObjectMemberOf,
		Read:   resourceReadObjectMemberOf,
		Update: resourceUpdateObjectMemberOf,
		Delete: resourceDeleteObjectMemberOf,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceCreateObjectMemberOf(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceCreateObjectMemberOf: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	objectDN := d.Get("object_dn").(string)
	groupDNs := d.Get("member_of").(*schema.Set)

	// make sure object exists
	entry, err := getObjectByDN(c.conn, objectDN)
	if err != nil {
		return fmt.Errorf("resourceCreateObjectMemberOf: unable to search object with dn:%v err:%w", objectDN, err)
	}
	rawGuid := entry.GetRawAttributeValue("objectGUID")
	guid, err := decodeGUID(rawGuid)
	if err != nil {
		return fmt.Errorf("resourceCreateObjectMemberOf: unable to convert raw GUID to string rawGUID:%x err:%w", rawGuid, err)
	}

	for _, groupDN := range groupDNs.List() {
		if err := validateDNString(c, groupDN.(string)); err != nil {
			return fmt.Errorf("resourceCreateObjectMemberOf: group dn is not valid err: %w", err)
		}
		if err := addObjectToGroup(c.conn, groupDN.(string), objectDN); err != nil {
			return fmt.Errorf("unable to add object to group: %s, err:%w", groupDN, err)
		}
	}

	// set GUID of group as resource ID
	d.SetId(guid)
	return resourceReadObjectMemberOf(d, meta)
}

func resourceReadObjectMemberOf(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceReadObjectMemberOf: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	id, err := encodeGUID(d.Id())
	if err != nil {
		return fmt.Errorf("resourceReadObjectMemberOf: unable to encode GUID:%v err:%w", d.Id(), err)
	}
	entry, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			c.logger.Error("resourceReadObjectMemberOf: object not found", "GUID", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceReadObjectMemberOf: unable to search object with ID dn:%v err:%w", d.Id(), err)
	}

	if err := updateObjectSchema(resourceActivedirectoryObjectMemberOf().Schema, entry, d); err != nil {
		return err
	}

	return nil
}

func resourceUpdateObjectMemberOf(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceUpdateObjectMemberOf: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()
	objectDN := d.Get("object_dn").(string)

	if d.HasChange("object_dn") {
		return fmt.Errorf("'activedirectory_object_memberof' will not make any changes to object DN. object_dn is only used to as reference.")
	}

	old, new := d.GetChange("member_of")
	oldGroups := old.(*schema.Set)
	newGroups := new.(*schema.Set)

	// get unique value from set
	uniqueNew := newGroups.Difference(oldGroups)
	uniqueOld := oldGroups.Difference(newGroups)
	for _, v := range uniqueNew.List() {
		if err := addObjectToGroup(c.conn, v.(string), objectDN); err != nil {
			return fmt.Errorf("resourceUpdateObjectMemberOf: unable to add object to group:%s, err:%w", v.(string), err)
		}
	}
	for _, v := range uniqueOld.List() {
		if err := removeObjectFromGroup(c.conn, v.(string), objectDN); err != nil {
			return fmt.Errorf("resourceUpdateObjectMemberOf: unable to remove object from group:%s, err:%w", v.(string), err)
		}
	}
	return resourceReadObjectMemberOf(d, meta)
}

func resourceDeleteObjectMemberOf(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceDeleteObjectMemberOf: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	objectDN := d.Get("object_dn").(string)
	groupDNs := d.Get("member_of").(*schema.Set)

	for _, groupDN := range groupDNs.List() {
		if err := removeObjectFromGroup(c.conn, groupDN.(string), objectDN); err != nil {
			return fmt.Errorf("unable to remove object from group: %s, err:%w", groupDN, err)
		}
	}

	c.logger.Info("resourceDeleteObjectMemberOf: AD object removed from groups", "dn", objectDN, "member_of", groupDNs.List())
	return nil
}

func addObjectToGroup(conn *ldap.Conn, groupDN, objectDN string) error {
	modReq := &ldap.ModifyRequest{DN: groupDN}
	modReq.Add("member", []string{objectDN})
	if err := conn.Modify(modReq); err != nil {
		// if we get result Code 68 "Entry Already Exists" return nil
		if ldap.IsErrorWithCode(err, 68) {
			return nil
		}
		return err
	}
	return nil
}

func removeObjectFromGroup(conn *ldap.Conn, groupDN, objectDN string) error {
	modReq := &ldap.ModifyRequest{DN: groupDN}
	modReq.Delete("member", []string{objectDN})
	if err := conn.Modify(modReq); err != nil {
		return err
	}
	return nil
}
