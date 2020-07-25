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
	resource.AddTestSweepers("activedirectory_ou", &resource.Sweeper{
		Name: "activedirectory_ou",
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

			sReq := &ldap.SearchRequest{
				BaseDN:       c.config.topDN,
				Scope:        ldap.ScopeWholeSubtree,
				DerefAliases: ldap.NeverDerefAliases,
				SizeLimit:    0,
				TimeLimit:    0,
				TypesOnly:    false,
				Filter:       "(ou=test_acc_ou*)",
				Attributes:   []string{"*"},
				Controls:     nil,
			}

			sr, err := c.conn.Search(sReq)
			if err != nil {
				if ldap.IsErrorWithCode(err, 32) {
					return nil
				}
				return err
			}
			var unDeleted []string
			for _, e := range sr.Entries {
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

func TestAccOU_Basic(t *testing.T) {
	base_ou := os.Getenv("AD_BASE_OU")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOUDestroy,
		Steps: []resource.TestStep{
			{
				// create object with only required argument defined
				Config: fmt.Sprintf(`resource "activedirectory_ou" "test_acc_ou1" {
					name             = "test_acc_ou1"
					base_ou_dn       = "%s"
				}`, base_ou),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_ou.test_acc_ou1"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou1", "name", "test_acc_ou1"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou1", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou1", "dn", "OU=test_acc_ou1,"+base_ou),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou1", "attributes", "{}"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou1", "description", ""),
				),
			},
		},
	})
}

func TestAccOU_Advanced(t *testing.T) {
	base_ou := os.Getenv("AD_BASE_OU")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOUDestroy,
		Steps: []resource.TestStep{
			{
				// create ou with optional arguments defined
				Config: testAccResourceADOUTestData("2", "test_acc_ou2", base_ou, "testing description"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_ou.test_acc_ou2"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "name", "test_acc_ou2"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "description", "testing description"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "dn", "OU=test_acc_ou2,"+base_ou),
				),
			}, {
				// change name description
				Config: testAccResourceADOUTestData("2", "test_acc_ou2_new", base_ou, "testing description update"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckObjectRemoteAttr("activedirectory_ou.test_acc_ou2"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "name", "test_acc_ou2_new"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "base_ou_dn", base_ou),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "description", "testing description update"),
					resource.TestCheckResourceAttr("activedirectory_ou.test_acc_ou2", "dn", "OU=test_acc_ou2_new,"+base_ou),
				),
			},
		},
	})
}

func testAccResourceADOUTestData(num, name, ou, description string) string {
	return fmt.Sprintf(`
resource "activedirectory_ou" "test_acc_ou%s" {
	name             = "%s"
	base_ou_dn       = "%s"
	description      = "%s"
}
`, num, name, ou, description)
}

func testAccCheckOUDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "activedirectory_ou" {
			continue
		}
		if err := isObjectDestroyed(rs); err != nil {
			return err
		}
	}
	return nil
}
