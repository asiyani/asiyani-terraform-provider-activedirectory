package activedirectory

import (
	"fmt"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	resource.AddTestSweepers("activedirectory_group", &resource.Sweeper{
		Name: "activedirectory_group",
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
			entries, err := getObjectsBySAM(c, "test_acc_group*")
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

func TestAccGroup_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGroupDestroy,
		Steps: []resource.TestStep{
			{
				// create object with only required argument defined
				Config: `resource "activedirectory_group" "test_acc_group1" {
					name             = "test_acc_group1"
					sam_account_name = "test_acc_group1"
					base_ou_dn       = "DC=dev,DC=private"
				}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_group.test_acc_group1"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "name", "test_acc_group1"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "sam_account_name", "test_acc_group1"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "dn", "CN=test_acc_group1,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "scope", "global"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "type", "security"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "attributes", "{}"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group1", "description", ""),
				),
			},
		},
	})
}

func TestAccGroup_Advanced(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGroupDestroy,
		Steps: []resource.TestStep{
			{
				// create group as global, security and other optional arguments defined
				Config: testAccResourceADGroupTestData("2", "test_acc_group2", "test_acc_group2",
					"DC=dev,DC=private", "testing description", "global", "security"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_group.test_acc_group2"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "name", "test_acc_group2"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "sam_account_name", "test_acc_group2"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "scope", "global"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "type", "security"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "dn", "CN=test_acc_group2,DC=dev,DC=private"),
				),
			}, {
				// change CN and OU
				Config: testAccResourceADGroupTestData("2", "test_acc_group2_update", "test_acc_group2",
					"CN=Builtin,DC=dev,DC=private", "testing description", "global", "security"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_group.test_acc_group2"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "name", "test_acc_group2_update"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "sam_account_name", "test_acc_group2"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "base_ou_dn", "CN=Builtin,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "scope", "global"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "type", "security"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group2", "dn", "CN=test_acc_group2_update,CN=Builtin,DC=dev,DC=private"),
				),
			}, {
				// create group as universal and distribution
				Config: testAccResourceADGroupTestData("3", "test_acc_group3", "test_acc_group3",
					"DC=dev,DC=private", "testing description", "universal", "distribution"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_group.test_acc_group3"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "name", "test_acc_group3"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "sam_account_name", "test_acc_group3"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "scope", "universal"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "type", "distribution"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group3", "dn", "CN=test_acc_group3,DC=dev,DC=private"),
				),
			}, {
				// create group as domain_local and security
				Config: testAccResourceADGroupTestData("4", "test_acc_group4", "test_acc_group4",
					"DC=dev,DC=private", "testing description", "domain_local", "security"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_group.test_acc_group4"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "name", "test_acc_group4"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "sam_account_name", "test_acc_group4"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "scope", "domain_local"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "type", "security"),
					resource.TestCheckResourceAttr("activedirectory_group.test_acc_group4", "dn", "CN=test_acc_group4,DC=dev,DC=private"),
				),
			},
		},
	})
}

func testAccResourceADGroupTestData(num, name, sam, ou, description, scope, groupType string) string {
	return fmt.Sprintf(`
resource "activedirectory_group" "test_acc_group%s" {
	name             = "%s"
	sam_account_name = "%s"
	base_ou_dn       = "%s"
	description      = "%s"
	scope            = "%s"
	type             = "%s"
}
`, num, name, sam, ou, description, scope, groupType)
}

func testAccCheckGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "activedirectory_group" {
			continue
		}
		if err := isObjectDestroyed(rs); err != nil {
			return err
		}
	}
	return nil
}
