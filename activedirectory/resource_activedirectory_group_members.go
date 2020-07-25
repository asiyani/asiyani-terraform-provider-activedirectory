package activedirectory

import (
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceActivedirectoryGroupMembers() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"group_dn": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The group's dn, to add AD object from members argument.",
				DiffSuppressFunc: ignoreCaseDiffSuppressor,
			},
			"members": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "List of object's dn to add to group.",
				Set:         lowercaseHashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
						v := val.(string)
						if _, err := ldap.ParseDN(v); err != nil {
							errs = append(errs, fmt.Errorf("member entry should be valid DN, got value:%s err:%v", v, err))
						}
						return
					},
				},
			},
		},
		Create: resourceCreateGroupMembers,
		Read:   resourceReadGroupMembers,
		Update: resourceUpdateGroupMembers,
		Delete: resourceDeleteGroupMembers,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceCreateGroupMembers(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceCreateGroupMembers: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()
	groupDN := d.Get("group_dn").(string)
	member := d.Get("members").(*schema.Set)
	if err := validateDNString(c, groupDN); err != nil {
		return fmt.Errorf("resourceCreateGroupMembers: group_dn is not valid err: %w", err)
	}

	// make sure group exists
	entry, err := getObjectByDN(c.conn, groupDN)
	if err != nil {
		return fmt.Errorf("resourceCreateGroupMembers: unable to search group with dn:%v err:%w", groupDN, err)
	}
	rawGuid := entry.GetRawAttributeValue("objectGUID")
	guid, err := decodeGUID(rawGuid)
	if err != nil {
		return fmt.Errorf("resourceCreateGroupMembers: unable to convert raw GUID to string rawGUID:%x err:%w", rawGuid, err)
	}

	var objectDN []string
	if member.Len() > 0 {
		for _, m := range member.List() {
			objectDN = append(objectDN, m.(string))
		}
	}

	// add each object just in case its already a member
	for _, o := range objectDN {
		modReq := &ldap.ModifyRequest{DN: groupDN}
		modReq.Add("member", []string{o})
		if err := c.conn.Modify(modReq); err != nil {
			// if we get result Code 68 "Entry Already Exists" we continue
			if ldap.IsErrorWithCode(err, 68) {
				continue
			}
			return fmt.Errorf("unable to add objects to group: %s, err:%w", groupDN, err)
		}
	}

	// set GUID of group as resource ID
	d.SetId(guid)
	return resourceReadGroupMembers(d, meta)
}

func resourceReadGroupMembers(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceReadGroupMembers: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	id, err := encodeGUID(d.Id())
	if err != nil {
		return fmt.Errorf("resourceReadGroupMembers: unable to encode GUID:%v err:%w", d.Id(), err)
	}
	entry, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			c.logger.Error("resourceReadGroupMembers: group not found", "GUID", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("resourceReadGroupMembers: unable to search group with ID:%v err:%w", d.Id(), err)
	}

	if err := updateObjectSchema(resourceActivedirectoryGroupMembers().Schema, entry, d); err != nil {
		return err
	}
	return nil
}

func resourceUpdateGroupMembers(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceUpdateGroupMembers: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	if d.HasChange("group_dn") {
		return fmt.Errorf("'activedirectory_group_members' will not make any changes to group DN. group_dn is only used to as reference.")
	}

	modReq := &ldap.ModifyRequest{DN: d.Get("group_dn").(string)}

	var replaceObj []string
	member := d.Get("members").(*schema.Set)
	for _, v := range member.List() {
		replaceObj = append(replaceObj, v.(string))
	}
	if len(replaceObj) > 0 {
		modReq.Replace("member", replaceObj)
	} else {
		modReq.Replace("member", []string{})
	}

	c.logger.Debug("resourceUpdateGroupMembers", "modify_request", modReq)

	if len(modReq.Changes) > 0 {
		if err = c.conn.Modify(modReq); err != nil {
			return fmt.Errorf("resourceUpdateGroupMembers: unable to update group membership err: %w", err)
		}
	}
	return resourceReadGroupMembers(d, meta)
}

func resourceDeleteGroupMembers(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceDeleteGroupMembers: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	groupDN := d.Get("group_dn").(string)
	member := d.Get("members").(*schema.Set)

	var objectDN []string
	if member.Len() > 0 {
		for _, m := range member.List() {
			objectDN = append(objectDN, m.(string))
		}
	}

	modReq := &ldap.ModifyRequest{DN: groupDN}
	modReq.Delete("member", objectDN)

	if err := c.conn.Modify(modReq); err != nil {
		return fmt.Errorf("unable to delete object from group: %s, err:%w", groupDN, err)
	}
	c.logger.Info("resourceDeleteGroupMembers: AD object removed from group", "dn", groupDN, "members", objectDN)
	return nil
}
