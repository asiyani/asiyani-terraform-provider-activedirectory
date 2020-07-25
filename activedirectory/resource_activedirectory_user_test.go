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
	base_ou := os.Getenv("AD_BASE_OU")
	domain := os.Getenv("AD_DOMAIN")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				// create object with only required argument defined
				Config: fmt.Sprintf(`resource "activedirectory_user" "test_acc_user1" {
					enabled             = false
					name                = "test_acc_user1"
					user_principal_name = "test_acc_user1@%s"
					sam_account_name    = "test_acc_user1$"
					base_ou_dn          = "%s"
				  }`, domain, base_ou),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user1"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "name", "test_acc_user1"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "sam_account_name", "test_acc_user1$"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "user_principal_name", "test_acc_user1@"+domain),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "enabled", "false"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user1", "dn", "CN=test_acc_user1,"+base_ou),
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
	base_ou := os.Getenv("AD_BASE_OU")
	domain := os.Getenv("AD_DOMAIN")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				// create object with all optional arguments defined
				Config: testAccResourceADUserTestData("2", "false", "John", "Doe", "John Doe", "test_acc_John.Doe", "John.Doe@"+domain, "secretPassword!123", base_ou, "testing description", `{company=["home"],department=["IT TF"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "false"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Doe"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Doe"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Doe"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Doe@"+domain),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Doe,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["home"],"department":["IT TF"]}`),
				),
			}, {
				// rename user and enable user
				Config: testAccResourceADUserTestData("2", "true", "John", "Smith", "John Smith", "test_acc_John.Smith", "John.Smith@"+domain, "secretPassword!123", base_ou, "testing description", `{company=["home"],department=["IT TF"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Smith@"+domain),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Smith,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["home"],"department":["IT TF"]}`),
				),
			}, {
				// update attributes and change password
				Config: testAccResourceADUserTestData("2", "true", "John", "Smith", "John Smith", "test_acc_John.Smith", "John.Smith@"+domain, "NewsecretPassword!123", base_ou, "testing description", `{company=["Terraform"],department=["IT"],departmentNumber=["24"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Smith@"+domain),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Smith,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "attributes", `{"company":["Terraform"],"department":["IT"],"departmentNumber":["24"]}`),
				),
			}, {
				// move user
				Config: testAccResourceADUserTestData("2", "true", "John", "Smith", "John Smith", "test_acc_John.Smith", "John.Smith@"+domain, "NewsecretPassword!123", "CN=Users,"+base_ou, "testing description", `{company=["Terraform"],department=["IT"],departmentNumber=["24"]}`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_user.test_acc_user2"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "first_name", "John"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "last_name", "Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "name", "John Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "sam_account_name", "test_acc_John.Smith"),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "user_principal_name", "John.Smith@"+domain),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "base_ou_dn", "CN=Users,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_user.test_acc_user2", "dn", "CN=John Smith,CN=Users,"+base_ou),
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
