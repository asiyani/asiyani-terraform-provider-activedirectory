package activedirectory

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func updateObjectSchema(resourceSchema map[string]*schema.Schema, e *ldap.Entry, d *schema.ResourceData) error {

	//range over resourceSchema to get attributes names
	for s := range resourceSchema {
		switch s {
		case "guid":
			rGUID := e.GetAttributeValue("objectGUID")
			guid, err := decodeGUID([]byte(rGUID))
			if err != nil {
				return fmt.Errorf("updateObjectSchema: unable to convert raw GUID to string rawGUID:%x err:%w", fmt.Sprintf("%x", rGUID), err)
			}
			if err = d.Set("guid", guid); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to set argument 'guid' value:%v err:%w", guid, err)
			}
		case "sid":
			// not all AD object have SID value, since its only used as data ignoring error
			rSid := e.GetAttributeValue("objectSid")
			sid, _ := decodeSID([]byte(rSid))
			// if err != nil {
			// 	return fmt.Errorf("updateObjectSchema: unable to convert raw SID to string rawSID:%x err:%w", fmt.Sprintf("%x", rSid), err)
			// }
			if err := d.Set("sid", sid); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to set argument 'sid' value: %v err: %w", sid, err)
			}
		case "cn":
			rcn := e.GetAttributeValue("cn")
			if err := d.Set("cn", rcn); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'cn' argument value: %v err: %w", rcn, err)
			}
		case "description":
			rdec := e.GetAttributeValue("description")
			if err := d.Set("description", rdec); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'description' argument value: %v err: %w", rdec, err)
			}
		case "sam_account_name":
			rsam := e.GetAttributeValue("sAMAccountName")
			if err := d.Set("sam_account_name", rsam); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'sam_account_name' argument value: %v err: %w", rsam, err)
			}
		case "name":
			rName := e.GetAttributeValue("name")
			if err := d.Set("name", rName); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'name' argument value:%v err:%w", rName, err)
			}
		case "dn":
			rDN := e.GetAttributeValue("distinguishedName")
			if err := d.Set("dn", rDN); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'dn' argument value:%v err:%w", rDN, err)
			}
		case "base_ou_dn":
			rDN := e.GetAttributeValue("distinguishedName")
			if err := d.Set("base_ou_dn", strings.SplitN(rDN, ",", 2)[1]); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'base_ou_dn' argument value:%v err:%w", strings.SplitN(rDN, ",", 2)[1], err)
			}
		case "user_account_control":
			ruac := e.GetAttributeValue("userAccountControl")
			if err := d.Set("user_account_control", ruac); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'user_account_control' argument value:%v err:%w", ruac, err)
			}
		case "enabled":
			ruac := e.GetAttributeValue("userAccountControl")
			status, err := isObjectEnabled(ruac)
			if err != nil {
				return fmt.Errorf("updateObjectSchema: unable parse userAccountControl to get Object enabled status argument value:%v err:%w", ruac, err)
			}
			if err := d.Set("enabled", status); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'enabled' argument value:%v err:%w", status, err)
			}
		case "attributes":
			currentAttr := map[string][]string{}
			newAttr := map[string][]string{}
			_ = json.Unmarshal([]byte(d.Get("attributes").(string)), &currentAttr)

			for name := range currentAttr {
				newAttr[name] = e.GetAttributeValues(name)
			}

			jsonNewAttr, err := json.Marshal(newAttr)
			if err != nil {
				return fmt.Errorf("updateObjectSchema: failed to marshal computer attributes to JSON, error: %s", err)
			}
			if err := d.Set("attributes", string(jsonNewAttr)); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'attributes' argument value:%v err:%w", jsonNewAttr, err)
			}

		// user object attributes
		case "first_name":
			rv := e.GetAttributeValue("givenName")
			if err := d.Set("first_name", rv); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'givenName' argument value:%v err:%w", rv, err)
			}
		case "last_name":
			rv := e.GetAttributeValue("sn")
			if err := d.Set("last_name", rv); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'sn' argument value:%v err:%w", rv, err)
			}
		case "user_principal_name":
			rv := e.GetAttributeValue("userPrincipalName")
			if err := d.Set("user_principal_name", rv); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'userPrincipalName' argument value:%v err:%w", rv, err)
			}

		// group object attributes
		case "scope", "type":
			gtv := e.GetAttributeValue("groupType")
			rs, rt, err := getGroupTypeScope(gtv)
			if err != nil {
				return fmt.Errorf("updateObjectSchema: unable to get group 'scope' and 'type' value from groupType value:%v err:%w", gtv, err)
			}
			if err := d.Set("scope", rs); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'scope' argument value:%v err:%w", rs, err)
			}
			if err := d.Set("type", rt); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'type' argument value:%v err:%w", rt, err)
			}
		case "members":
			memValues := e.GetAttributeValues("member")
			mem := make([]interface{}, len(memValues))
			for i, v := range memValues {
				mem[i] = v
			}
			memSet := schema.NewSet(lowercaseHashString, mem)
			if err := d.Set("members", memSet); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'members' argument value:%v err:%w", memValues, err)
			}
		case "member_of":
			memOfValues := e.GetAttributeValues("memberOf")
			memOf := make([]interface{}, len(memOfValues))
			for i, v := range memOfValues {
				memOf[i] = v
			}
			memOfSet := schema.NewSet(lowercaseHashString, memOf)
			if err := d.Set("member_of", memOfSet); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'member_of' argument value:%v err:%w", memOfValues, err)
			}

		// OU object attributes
		case "ou":
			rv := e.GetAttributeValue("ou")
			if err := d.Set("ou", rv); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'ou' argument value:%v err:%w", rv, err)
			}

		// group membership attributes
		case "group_dn":
			rDN := e.GetAttributeValue("distinguishedName")
			if err := d.Set("group_dn", rDN); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'dn' argument value:%v err:%w", rDN, err)
			}
		// object memberof attributes
		case "object_dn":
			rDN := e.GetAttributeValue("distinguishedName")
			if err := d.Set("object_dn", rDN); err != nil {
				return fmt.Errorf("updateObjectSchema: unable to update 'dn' argument value:%v err:%w", rDN, err)
			}

		}
	}

	return nil
}
