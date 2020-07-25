package activedirectory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"activedirectory": testAccProvider,
	}
}
func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AD_LDAP_URL"); v == "" {
		t.Fatal("AD_LDAP_URL must be set for acceptance tests")
	}
	if v := os.Getenv("AD_DOMAIN"); v == "" {
		t.Fatal("AD_DOMAIN must be set for acceptance tests")
	}
	if v := os.Getenv("AD_BIND_USERNAME"); v == "" {
		t.Fatal("AD_BIND_USERNAME must be set for acceptance tests")
	}
	if v := os.Getenv("AD_BIND_PASSWORD"); v == "" {
		t.Fatal("AD_BIND_PASSWORD must be set for acceptance tests")
	}
}

// helper function for all test to check remote object attributes
func testAccCheckObjectRemoteAttr(resource string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resource]
		if !ok {
			return fmt.Errorf("Not found: %s", resource)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}
		oID := rs.Primary.ID

		c := testAccProvider.Meta().(*ADClient)
		err := c.initialiseConn()
		if err != nil {
			return fmt.Errorf("unable to connect to LDAP server err:%w", err)
		}
		defer c.done()

		id, err := encodeGUID(oID)
		if err != nil {
			return fmt.Errorf("unable to encode GUID:%v err:%w", id, err)
		}
		e, err := getObjectByID(c, id)
		if err != nil {
			return fmt.Errorf("error fetching AD object with resource %s. %s", resource, err)
		}

		// also check attributes of remote object
		for k, v := range rs.Primary.Attributes {
			switch k {
			case "dn":
				rv := e.GetAttributeValue("distinguishedName")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("distinguishedName in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "cn":
				rv := e.GetAttributeValue("cn")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("cn in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "name":
				rv := e.GetAttributeValue("name")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("name in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "base_ou_dn":
				dn := e.GetAttributeValue("distinguishedName")
				rv := strings.SplitN(dn, ",", 2)[1]
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("distinguishedName in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "enabled":
				rv := e.GetAttributeValue("userAccountControl")
				status, err := isObjectEnabled(rv)
				if err != nil {
					return fmt.Errorf("unable parse userAccountControl to get Object enabled status argument value:%v err:%w", rv, err)
				}
				if v != strconv.FormatBool(status) {
					return fmt.Errorf("enabled in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "sam_account_name":
				rv := e.GetAttributeValue("sAMAccountName")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("sAMAccountName in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "description":
				rv := e.GetAttributeValue("description")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("description in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "attributes":
				// value of attributes is actually a json of map[string][]string
				attrMap := map[string][]string{}
				_ = json.Unmarshal([]byte(v), &attrMap)
				// loop through json map and check remote attributes value.
				for jak, jav := range attrMap {
					rv := e.GetAttributeValues(jak)
					if !compareAttrValues(rv, jav) {
						return fmt.Errorf("attributes:%s in state and remote object is different.  state:%s, Remote:%s", jak, jav, rv)
					}
				}
			// user object attributes
			case "first_name":
				rv := e.GetAttributeValue("givenName")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("givenName(first_name) in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "last_name":
				rv := e.GetAttributeValue("sn")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("sn(last_name) in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "user_principal_name":
				rv := e.GetAttributeValue("userPrincipalName")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("userPrincipalName in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			// group object attributes
			case "scope":
				gtv := e.GetAttributeValue("groupType")
				rv, _, err := getGroupTypeScope(gtv)
				if err != nil {
					return fmt.Errorf("unable to get group 'scope' and 'type' value from groupType value:%v err:%w", gtv, err)
				}
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("groupType-scope in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			case "type":
				gtv := e.GetAttributeValue("groupType")
				_, rv, err := getGroupTypeScope(gtv)
				if err != nil {
					return fmt.Errorf("unable to get group 'scope' and 'type' value from groupType value:%v err:%w", gtv, err)
				}
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("groupType-type in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			// OU object attributes
			case "ou":
				rv := e.GetAttributeValue("ou")
				if !strings.EqualFold(v, rv) {
					return fmt.Errorf("ou in state and remote object is different.  state:%s, Remote:%s", v, rv)
				}
			}
		}
		return nil
	}
}

func isObjectDestroyed(rs *terraform.ResourceState) error {
	c := testAccProvider.Meta().(*ADClient)
	err := c.initialiseConn()
	if err != nil {
		return fmt.Errorf("unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	oID := rs.Primary.ID
	id, err := encodeGUID(oID)
	if err != nil {
		return fmt.Errorf("unable to encode GUID:%v err:%w", id, err)
	}
	e, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return nil
		}
		return fmt.Errorf("unable to search for AD object with ID:%s, err:%w", oID, err)
	}
	if e != nil {
		return fmt.Errorf("ad object (%s) still exists", rs.Primary.ID)
	}
	return nil
}
