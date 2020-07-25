# activedirectory_object_memberof

This resource allows to create and configure an Active Directory object membership with groups. Objects or group doesn't have to be managed by TF.

## Example Usage

```hcl
resource "activedirectory_computer" "server1" {
	name             = "server1"
	sam_account_name = "server1$"
	base_ou_dn       = "OU=Servers,DC=exmample,DC=com"
}
resource "activedirectory_group" "group1" {
	name             = "group1"
	sam_account_name = "group1"
	base_ou_dn       = "OU=Groups,DC=exmample,DC=com"
}

# this resource will add server1 to group1 & 'Application Server' group
resource "activedirectory_object_memberof" "server1_memberof" {
	object_dn = activedirectory_computer.server1.dn
	member_of = [
		activedirectory_group.group1.dn, 
		"cn=Application Server,OU=Groups,OU=Groups,DC=exmample,DC=com"
		]
}
```

## Argument Reference

* `object_dn` - (Required) - The AD object's dn to add in groups, should be of computer, user or group dn.
* `member_of` - (Required) - List of group's dns to add AD Object.

## Import

This resource can be imported using active directory ObjectGUID of the object.

`$ terraform import activedirectory_object_memberof.example <object_objectGUID>`
