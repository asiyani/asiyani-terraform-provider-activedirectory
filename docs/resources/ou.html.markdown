# activedirectory_ou

This resource allows you to create and configure an Active Directory organizational unit.

## Example Usage

```hcl
# basic example
resource "activedirectory_ou" "workstations" {
    name             = "workstations"
    base_ou_dn       = "OU=Resources,DC=example,DC=com"
}

resource "activedirectory_ou" "servers" {
	name             = "servers"
	base_ou_dn       = "OU=Resources,DC=example,DC=com"
	description      = "OU created and maintained via terraform"
}
```

## Argument Reference

* `name` - (Required) - The name of the Object.
* `base_ou_dn` - (Required) - The `dn` (distinguished name) of the `OU` (Organizational Unit) or container where the object is created.

* `description` - (Optional) - A description for the AD object.
* `attributes` - (Optional) - The list of other attributes of object, represented in json as map with `attribute name` as key and values as array of string ie `{attribute_name = ["value1","value2"]}`.

##  Attributes Reference

* `ou` - The name property of the OU.
* `dn` - The distinguished name (dn) of the object.
* `guid` - The ``ObjectGUID of the object. value is in hexadecimal format and in Endian Ordering used by Microsoft Active Directory.

## Import

This resource can be imported using active directory ObjectGUID of the object.

`$ terraform import activedirectory_ou.example <ObjectGUID>`

example

`$ terraform import activedirectory_ou.workstations e6e2b065-5a82-43bc-9fdb-6ec491de3d1d`