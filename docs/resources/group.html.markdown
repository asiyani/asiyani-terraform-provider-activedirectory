# activedirectory_group

This resource allows you to create and configure an Active Directory Group.

## Example Usage

```hcl
# basic example
resource "activedirectory_group" "group1" {
    name             = "group1"
    sam_account_name = "group1"
    base_ou_dn       = "OU=Groups,OU=Resources,DC=example,DC=com"
}

resource "activedirectory_group" "group2" {
	name             = "group2"
	sam_account_name = "group2"
	base_ou_dn       = "OU=Groups,OU=Resources,DC=example,DC=com"
	description      = "group created and maintained via terraform"
	scope            = "global"
	type             = "security"
}
```

## Argument Reference

* `name` - (Required) - The name of the Object.
* `base_ou_dn` - (Required) - The `dn` (distinguished name) of the `OU` (Organizational Unit) or container where the object is created.
* `sam_account_name` - (Required) - The sAMAccountName attribute of the object. It must be 20 or fewer characters.

* `scope` - (Optional) - The group scope, allowed values are `domain_local`,`global` and `universal`, default value `global`.
* `type` - (Optional) - The group type, allowed values are `security` and `distribution`, default value `security`.
* `description` - (Optional) - A description for the AD object.
* `attributes` - (Optional) - The list of other attributes of object, represented in json as map with `attribute name` as key and values as array of string ie `{attribute_name = ["value1","value2"]}`.

##  Attributes Reference

* `cn` - The Common-Name property of the object.
* `dn` - The distinguished name (dn) of the object.
* `guid` - The `ObjectGUID` of the object. value is in hexadecimal format and in Endian Ordering used by Microsoft Active Directory.
* `sid` - The security identifier (SID) of the object.
* `members` - The member attribute of the AD object. contains object's DN.
* `member_of` - The memberOf attribute of the AD object. contains object's DN.

## Import

This resource can be imported using active directory ObjectGUID of the object.

`$ terraform import activedirectory_group.example <ObjectGUID>`

example

`$ terraform import activedirectory_group.group1 e6e2b065-5a82-43bc-9fdb-6ec491de3d1d`