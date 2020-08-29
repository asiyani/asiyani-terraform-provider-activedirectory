package activedirectory

import (
	"fmt"
	"os"
	"strconv"
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
	baseOU := os.Getenv("AD_BASE_OU")
	member1DN := "CN=test_acc_comp1," + baseOU
	member2DN := "CN=test_acc_comp2," + baseOU
	member3DN := "CN=test_acc_comp3," + baseOU
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGroupMemberDestroy,
		Steps: []resource.TestStep{
			{
				// add 2 computer to a group
				Config: testAccResourceADGroupMembersTestData(baseOU, `[activedirectory_computer.test_acc_comp1.dn, activedirectory_computer.test_acc_comp2.dn]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGroupMemberRemoteAttr(baseOU, "activedirectory_group_members.test_acc_group_member"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "group_dn", "CN=test_acc_group1,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members."+strconv.Itoa(lowercaseHashString(member1DN)), member1DN),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members."+strconv.Itoa(lowercaseHashString(member2DN)), member2DN),
				),
			}, {
				// remove 1 and add another computer to a group
				Config: testAccResourceADGroupMembersTestData(baseOU, `[activedirectory_computer.test_acc_comp1.dn, activedirectory_computer.test_acc_comp3.dn]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGroupMemberRemoteAttr(baseOU, "activedirectory_group_members.test_acc_group_member"),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "group_dn", "CN=test_acc_group1,"+baseOU),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members."+strconv.Itoa(lowercaseHashString(member1DN)), member1DN),
					resource.TestCheckNoResourceAttr("activedirectory_group_members.test_acc_group_member", "members."+strconv.Itoa(lowercaseHashString(member2DN))),
					resource.TestCheckResourceAttr("activedirectory_group_members.test_acc_group_member", "members."+strconv.Itoa(lowercaseHashString(member3DN)), member3DN),
				),
			},
		},
	})
}

// also create 3 computer and 1 group resource to test membership
func testAccResourceADGroupMembersTestData(baseOU, members string) string {
	return fmt.Sprintf(`
resource "activedirectory_computer" "test_acc_comp1" {
	name             = "test_acc_comp1"
	sam_account_name = "test_acc_comp1$"
	base_ou_dn       = "%s"
}

resource "activedirectory_computer" "test_acc_comp2" {
	name             = "test_acc_comp2"
	sam_account_name = "test_acc_comp2$"
	base_ou_dn       = "%s"
}

resource "activedirectory_computer" "test_acc_comp3" {
	name             = "test_acc_comp3"
	sam_account_name = "test_acc_comp3$"
	base_ou_dn       = "%s"
}

resource "activedirectory_group" "test_acc_group1" {
	name             = "test_acc_group1"
	sam_account_name = "test_acc_group1"
	base_ou_dn       = "%s"
}
resource "activedirectory_group_members" "test_acc_group_member" {
	group_dn = activedirectory_group.test_acc_group1.dn
	members  = %s
}
`, baseOU, baseOU, baseOU, baseOU, members)
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
func testAccCheckGroupMemberRemoteAttr(baseOU, resource string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		member1DN := "CN=test_acc_comp1," + baseOU
		member2DN := "CN=test_acc_comp2," + baseOU
		member3DN := "CN=test_acc_comp3," + baseOU
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
			case "members." + strconv.Itoa(lowercaseHashString(member1DN)):
				if !contains(remoteMembers, v) {
					return fmt.Errorf("member.%s in state and remote object is different.  state:%s, Remote:%s", strconv.Itoa(lowercaseHashString(member1DN)), v, remoteMembers)
				}
			case "members." + strconv.Itoa(lowercaseHashString(member2DN)):
				if !contains(remoteMembers, v) {
					return fmt.Errorf("member.%s in state and remote object is different.  state:%s, Remote:%s", strconv.Itoa(lowercaseHashString(member2DN)), v, remoteMembers)
				}
			case "members." + strconv.Itoa(lowercaseHashString(member3DN)):
				if !contains(remoteMembers, v) {
					return fmt.Errorf("member.%s in state and remote object is different.  state:%s, Remote:%s", strconv.Itoa(lowercaseHashString(member3DN)), v, remoteMembers)
				}
			default:
				if strings.HasPrefix(k, "members") && k != "members.#" {
					return fmt.Errorf("unknown members attribute found in state, key: %s. value: %s\n", k, v)
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
