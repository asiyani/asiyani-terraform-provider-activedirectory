package activedirectory

import (
	"fmt"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	resource.AddTestSweepers("activedirectory_user", &resource.Sweeper{
		Name: "activedirectory_user",
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
			entries, err := getObjectsBySAM(c, "test_acc_user*")
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

func TestAccUser_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				// create object with only required argument defined
				Config: `resource "activedirectory_user" "test_acc_user1" {
					enabled             = false
					name                = "test_acc_user1"
					user_principal_name = "test_acc_user1@dev.private"
					sam_account_name    = "test_acc_user1$"
					base_ou_dn          = "DC=dev,DC=private"
				  }`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user1"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "name", "test_acc_user1"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "sam_account_name", "test_acc_user1$"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "user_principal_name", "test_acc_user1@dev.private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "enabled", "false"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "dn", "CN=test_acc_user1,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "attributes", "{}"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "description", ""),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "first_name", ""),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "last_name", ""),
					resource.TestCheckNoResourceAttr("activedirectory_user.test_acc_user1", "password"),
				),
			},
		},
	})
}

func TestAccUser_Advanced(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				// create object with all optional arguments defined
				Config: testAccResourceADUserTestData("2", "false", "John", "Doe", "John Doe", "test_acc_John.Doe", "John.Doe@dev.private", "secretPassword!123", "DC=dev,DC=private", "testing description", `{company=["home"],department=["IT TF"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "false"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Doe"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Doe"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Doe"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Doe@dev.private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Doe,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["home"],"department":["IT TF"]}`),
				),
			}, {
				// rename user and enable user
				Config: testAccResourceADUserTestData("2", "true", "John", "Smith", "John Smith", "test_acc_John.Smith", "John.Smith@dev.private", "secretPassword!123", "DC=dev,DC=private", "testing description", `{company=["home"],department=["IT TF"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Smith@dev.private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Smith,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["home"],"department":["IT TF"]}`),
				),
			}, {
				// update attributes and change password
				Config: testAccResourceADUserTestData("2", "true", "John", "Smith", "John Smith", "test_acc_John.Smith", "John.Smith@dev.private", "NewsecretPassword!123", "DC=dev,DC=private", "testing description", `{company=["Terraform"],department=["IT"],departmentNumber=["24"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Smith@dev.private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", "DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Smith,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["Terraform"],"department":["IT"],"departmentNumber":["24"]}`),
				),
			}, {
				// move user
				Config: testAccResourceADUserTestData("2", "true", "John", "Smith", "John Smith", "test_acc_John.Smith", "John.Smith@dev.private", "NewsecretPassword!123", "CN=Users,DC=dev,DC=private", "testing description", `{company=["Terraform"],department=["IT"],departmentNumber=["24"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Smith@dev.private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", "CN=Users,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Smith,CN=Users,DC=dev,DC=private"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["Terraform"],"department":["IT"],"departmentNumber":["24"]}`),
				),
			},
		},
	})
}

func testAccResourceADUserTestData(num, enabled, first, last, name, sam, upn, pass, ou, description, attributes string) string {
	return fmt.Sprintf(`
resource "activedirectory_user" "test_acc_user%s" {
	enabled             = %s
	first_name          = "%s"
	last_name           = "%s"
	name                = "%s"
	sam_account_name    = "%s"
	user_principal_name = "%s"
	password            = "%s"
	base_ou_dn          = "%s"
	description         = "%s"
	attributes          = jsonencode(%s)
}`, num, enabled, first, last, name, sam, upn, pass, ou, description, attributes)
}

func testAccCheckUserDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "activedirectory_user" {
			continue
		}
		if err := isObjectDestroyed(rs); err != nil {
			return err
		}
	}
	return nil
}
