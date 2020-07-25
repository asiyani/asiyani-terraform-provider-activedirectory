# activedirectory_group_members

This resource allows to create and configure an Active Directory group membership with objects. Objects or group doesn't have to be managed by TF.

## Example Usage

```hcl
resource "activedirectory_computer" "server1" {
    name             = "server1"
    sam_account_name = "server1$"	
    base_ou_dn       = "OU=Servers,OU=Resources,DC=example,DC=com"
}
resource "activedirectory_computer" "server2" {
	name             = "server2"
	sam_account_name = "server2$"
	base_ou_dn       = "OU=Servers,OU=Resources,DC=example,DC=com"
}

# this resource will add server1 and server2 to group 'group1'
resource "activedirectory_group_members" "group1_members" {
	group_dn = "CN=group1,OU=Groups,OU=Resources,DC=example,DC=com"
	members  = [
        activedirectory_computer.server1.dn, 
        "cn=server2,OU=Workstations,OU=Resources,DC=example,DC=com"
        ]
}
```

## Argument Reference

* `group_dn` - (Required) - The dn of the group you want to add members to.
* `members` - (Required) - List of object's dns to add to group.

## Import

This resource can be imported using active directory ObjectGUID of the object.

`$ terraform import activedirectory_group_members.example <group_objectGUID>`
