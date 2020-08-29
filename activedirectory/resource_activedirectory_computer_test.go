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
	resource.AddTestSweepers("activedirectory_computer", &resource.Sweeper{
		Name: "activedirectory_computer",
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
			entries, err := getObjectsBySAM(c, "test_acc_comp*")
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

func TestAccComputer_Basic(t *testing.T) {
	baseOU := os.Getenv("AD_BASE_OU")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputerDestroy,
		Steps: []resource.TestStep{
			{
				// create object with only required argument defined
				Config: fmt.Sprintf(`resource "activedirectory_computer" "test_acc_comp1" {
					name             = "test_acc_comp1"
					sam_account_name = "test_acc_comp1$"	
					base_ou_dn       = "%s"
				}`, baseOU),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_computer.test_acc_comp1"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "name", "test_acc_comp1"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "sam_account_name", "test_acc_comp1$"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "base_ou_dn", baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "dn", "CN=test_acc_comp1,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "attributes", "{}"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp1", "description", ""),
				),
			},
		},
	})
}

func TestAccComputer_Advanced(t *testing.T) {
	baseOU := os.Getenv("AD_BASE_OU")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputerDestroy,
		Steps: []resource.TestStep{
			{
				// create object with all optional arguments defined
				Config: testAccResourceADComputerTestData("2", "false", "test_acc_comp2", "test_acc_comp2",
					baseOU, "testing description", `{company=["home"],department=["IT TF"]}`, baseOU),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_computer.test_acc_comp2"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "name", "test_acc_comp2"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "sam_account_name", "test_acc_comp2$"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "enabled", "false"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "base_ou_dn", baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "dn", "CN=test_acc_comp2,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "attributes", `{"company":["home"],"department":["IT TF"]}`),
				),
			}, {
				// enabled object, Change CN, OU and attributes
				Config: testAccResourceADComputerTestData("2", "true", "test_acc_comp2_update", "test_acc_comp2",
					"${activedirectory_ou.test_acc_ou_cmpMove.dn}", "testing description", `{company=["Terraform"],department=["IT"],departmentNumber=["24"]}`, baseOU),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_computer.test_acc_comp2"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "name", "test_acc_comp2_update"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "sam_account_name", "test_acc_comp2$"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "base_ou_dn", "OU=test_acc_ou_cmpMove,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "dn", "CN=test_acc_comp2_update,OU=test_acc_ou_cmpMove,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "attributes", `{"company":["Terraform"],"department":["IT"],"departmentNumber":["24"]}`),
				),
			}, {
				// changed SAM name and remove some attributes
				Config: testAccResourceADComputerTestData("2", "true", "test_acc_comp2_update", "test_acc_comp2_new",
					"${activedirectory_ou.test_acc_ou_cmpMove.dn}", "testing description", `{company=["Terraform"]}`, baseOU),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_computer.test_acc_comp2"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "name", "test_acc_comp2_update"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "sam_account_name", "test_acc_comp2_new$"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "enabled", "true"),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "base_ou_dn", "OU=test_acc_ou_cmpMove,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "dn", "CN=test_acc_comp2_update,OU=test_acc_ou_cmpMove,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_computer.test_acc_comp2", "attributes", `{"company":["Terraform"]}`),
				),
			},
		},
	})
}

func testAccResourceADComputerTestData(num, enabled, name, sam, ou, description, attributes, baseOU string) string {
	return fmt.Sprintf(`
resource "activedirectory_computer" "test_acc_comp%s" {
	enabled          = %s
	name             = "%s"
	sam_account_name = "%s$"	
	base_ou_dn       = "%s"
	description      = "%s"
	attributes = jsonencode(%s)
}

resource "activedirectory_ou" "test_acc_ou_cmpMove" {
	name             = "test_acc_ou_cmpMove"
	base_ou_dn       = "%s"
}
`, num, enabled, name, sam, ou, description, attributes, baseOU)
}

func testAccCheckComputerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "activedirectory_computer" {
			continue
		}
		if err := isObjectDestroyed(rs); err != nil {
			return err
		}
	}
	return nil
}
