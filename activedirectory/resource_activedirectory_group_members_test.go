package activedirectory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	resource.AddTestSweepers("activedirectory_group_members", &resource.Sweeper{
		Name: "activedirectory_group_members",
		F: func(r string) error {
			client, err := sharedClient()
			if err != nil {
				return fmt.Errorf("Error getting client: %s", err)
			}
			c := client.(*ADClient)

			err = c.initialiseConn()
			if err != nil {
				return fmt.Errorf("unable to connect to LDAP server err:%w", err)
			}
			defer c.done()
			entries, err := getObjectsBySAM(c, "test_acc_*")
			if err != nil {
				return err
			}
			var unDeleted []string
			for _, e := range entries {
				c.logger.Info("Sweep test deleting object...", "DN", e.DN)
				request := ldap.DelRequest{DN: e.DN}
				err = c.conn.Del(&request)
				if err != nil {
					c.logger.Error("unable to delete object", "DN", e.DN, "err", err)
					unDeleted = append(unDeleted, e.DN)
				}
			}
			if len(unDeleted) != 0 {
				return fmt.Errorf("unable to delete object, DNs: %s", unDeleted)
			}
			return nil
		},
	})
}

func TestAccGroupMembers_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGroupMemberDestroy,
		Steps: []resource.TestStep{
			{
				// add 2 computer to a group
				Config: testAccResourceADGroupMembersTestData(`[activedirectory_computer.test_acc_comp1.dn, activedirectory_computer.test_acc_comp2.dn]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGroupMemberRemoteAttr("activedirectory_group_members.test_acc_group_member"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "group_dn", "CN=test_acc_group1,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members.2678311993", "CN=test_acc_comp1,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members.3837611738", "CN=test_acc_comp2,DC=dev,DC=private"),
				),
			}, {
				// remove 1 and add another computer to a group
				Config: testAccResourceADGroupMembersTestData(`[activedirectory_computer.test_acc_comp1.dn, activedirectory_computer.test_acc_comp3.dn]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGroupMemberRemoteAttr("activedirectory_group_members.test_acc_group_member"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "group_dn", "CN=test_acc_group1,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members.2678311993", "CN=test_acc_comp1,DC=dev,DC=private"),
					resource.TestCheckNoResourceAttr("activedirectory_group_members.test_acc_group_member", "members.3837611738"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members.2070400324", "CN=test_acc_comp3,DC=dev,DC=private"),
				),
			},
		},
	})
}

// also create 3 computer and 1 group resource to test membership
func testAccResourceADGroupMembersTestData(members string) string {
	return fmt.Sprintf(`
resource "activedirectory_computer" "test_acc_comp1" {
	name             = "test_acc_comp1"
	sam_account_name = "test_acc_comp1$"
	base_ou_dn       = "DC=dev,DC=private"
}

resource "activedirectory_computer" "test_acc_comp2" {
	name             = "test_acc_comp2"
	sam_account_name = "test_acc_comp2$"
	base_ou_dn       = "DC=dev,DC=private"
}

resource "activedirectory_computer" "test_acc_comp3" {
	name             = "test_acc_comp3"
	sam_account_name = "test_acc_comp3$"
	base_ou_dn       = "DC=dev,DC=private"
}

resource "activedirectory_group" "test_acc_group1" {
	name             = "test_acc_group1"
	sam_account_name = "test_acc_group1"
	base_ou_dn       = "DC=dev,DC=private"
}
resource "activedirectory_group_members" "test_acc_group_member" {
	group_dn = activedirectory_group.test_acc_group1.dn
	members  = %s
}
`, members)
}

func testAccCheckGroupMemberDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if err := isObjectDestroyed(rs); err != nil {
			return err
		}
	}
	return nil
}

// helper function for all test to check remote object attributes
func testAccCheckGroupMemberRemoteAttr(resource string) resource.TestCheckFunc {
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
		remoteMembers := e.GetAttributeValues("member")

		// also check attributes of remote object
		for k, v := range rs.Primary.Attributes {
			switch k {
			case "members.2678311993":
				if !contains(remoteMembers, v) {
					return fmt.Errorf("member.2678311993 in state and remote object is different.  state:%s, Remote:%s", v, remoteMembers)
				}
			case "members.3837611738":
				if !contains(remoteMembers, v) {
					return fmt.Errorf("member.3837611738 in state and remote object is different.  state:%s, Remote:%s", v, remoteMembers)
				}
			case "members.2070400324":
				if !contains(remoteMembers, v) {
					return fmt.Errorf("member.2070400324 in state and remote object is different.  state:%s, Remote:%s", v, remoteMembers)
				}
			}
		}
		return nil
	}
}

func contains(values []string, val string) bool {
	for _, v := range values {
		if strings.EqualFold(v, val) {
			return true
		}
	}
	return false
}
