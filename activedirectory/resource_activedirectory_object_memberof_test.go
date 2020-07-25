package activedirectory

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	resource.AddTestSweepers("activedirectory_object_memberof", &resource.Sweeper{
		Name: "activedirectory_object_memberof",
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

func TestAccObjectMemberOf_Basic(t *testing.T) {
	base_ou := os.Getenv("AD_BASE_OU")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckObjectMemberOfDestroy,
		Steps: []resource.TestStep{
			{
				// add 2 computer to a group
				Config: testAccResourceObjectMemberOfTestData(base_ou, `[activedirectory_group.test_acc_group1.dn, activedirectory_group.test_acc_group2.dn]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckObjectMemberOfRemoteAttr("activedirectory_object_memberof.test_acc_obj_memberof"),
					resource.TestCheckResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "object_dn", "CN=test_acc_comp1,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "member_of.2656686201", "CN=test_acc_group1,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "member_of.3846656666", "CN=test_acc_group2,"+base_ou),
				),
			}, {
				// remove 1 and add another computer to a group
				Config: testAccResourceObjectMemberOfTestData(base_ou, `[activedirectory_group.test_acc_group1.dn, activedirectory_group.test_acc_group3.dn]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckObjectMemberOfRemoteAttr("activedirectory_object_memberof.test_acc_obj_memberof"),
					resource.TestCheckResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "object_dn", "CN=test_acc_comp1,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "member_of.2656686201", "CN=test_acc_group1,"+base_ou),
					resource.TestCheckNoResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "member_of.3846656666"),
					resource.TestCheckResourceAttr("activedirectory_object_memberof.test_acc_obj_memberof", "member_of.2057163012", "CN=test_acc_group3,"+base_ou),
				),
			},
		},
	})
}

// also create 1 computer and 3 group resource to test membership
func testAccResourceObjectMemberOfTestData(base_ou, members string) string {
	return fmt.Sprintf(`
resource "activedirectory_computer" "test_acc_comp1" {
	name             = "test_acc_comp1"
	sam_account_name = "test_acc_comp1$"
	base_ou_dn       = "%s"
}
resource "activedirectory_group" "test_acc_group1" {
	name             = "test_acc_group1"
	sam_account_name = "test_acc_group1"
	base_ou_dn       = "%s"
}
resource "activedirectory_group" "test_acc_group2" {
	name             = "test_acc_group2"
	sam_account_name = "test_acc_group2"
	base_ou_dn       = "%s"
}
resource "activedirectory_group" "test_acc_group3" {
	name             = "test_acc_group3"
	sam_account_name = "test_acc_group3"
	base_ou_dn       = "%s"
}
resource "activedirectory_object_memberof" "test_acc_obj_memberof" {
	object_dn = activedirectory_computer.test_acc_comp1.dn
	member_of = %s
}
`, base_ou, base_ou, base_ou, base_ou, members)
}

func testAccCheckObjectMemberOfDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if err := isObjectDestroyed(rs); err != nil {
			return err
		}
	}
	return nil
}

// helper function for all test to check remote object attributes
func testAccCheckObjectMemberOfRemoteAttr(resource string) resource.TestCheckFunc {
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
		remoteMemberOf := e.GetAttributeValues("memberOf")

		// also check attributes of remote object
		for k, v := range rs.Primary.Attributes {
			switch k {
			case "member_of.2656686201":
				if !contains(remoteMemberOf, v) {
					return fmt.Errorf("member_of.2656686201 in state and remote object is different.  state:%s, Remote:%s", v, remoteMemberOf)
				}
			case "member_of.3846656666":
				if !contains(remoteMemberOf, v) {
					return fmt.Errorf("member_of.3846656666 in state and remote object is different.  state:%s, Remote:%s", v, remoteMemberOf)
				}
			case "member_of.2057163012":
				if !contains(remoteMemberOf, v) {
					return fmt.Errorf("member_of.2057163012 in state and remote object is different.  state:%s, Remote:%s", v, remoteMemberOf)
				}
			}
		}
		return nil
	}
}
