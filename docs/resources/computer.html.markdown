# activedirectory_computer

This resource allows you to create and configure an Active Directory Computer.

## Example Usage

```hcl
# basic example
resource "activedirectory_computer" "desktop1" {
    name             = "desktop1"
    sam_account_name = "desktop1$"	
    base_ou_dn       = "OU=Workstations,OU=Resources,DC=example,DC=com"
}

resource "activedirectory_computer" "desktop2" {
  name             = "desktop2"
  enabled          = true
  sam_account_name = "desktop2$"
  base_ou_dn       = "OU=Workstations,OU=Resources,DC=example,DC=com"
  description      = "desktop created and maintained via terraform"
  attributes = jsonencode({
    department = ["Sales"],
    company    = ["example"]
  })
}
```

## Argument Reference

* `name` - (Required) - The name of the Object.
* `base_ou_dn` - (Required) - The `dn` (distinguished name) of the `OU` (Organizational Unit) or container where the object is created.
* `sam_account_name` - (Required) - The sAMAccountName attribute of the object. It must be 20 or fewer characters and for computer object it must end with `$`.

* `enabled` - (Optional) - The enabled status of Object, default is true.
* `description` - (Optional) - A description for the AD object.
* `attributes` - (Optional) - The list of other attributes of object, represented in json as map with `attribute name` as key and values as array of string ie `{attribute_name = ["value1","value2"]}`.

##  Attributes Reference

* `cn` - The Common-Name property of the object.
* `dn` - The distinguished name (dn) of the object.
* `guid` - The `ObjectGUID` of the object. value is in hexadecimal format and in Endian Ordering used by Microsoft Active Directory.
* `sid` - The security identifier (SID) of the object.
* `user_account_control` - The userAccountControl Attribute Flags that control the behaviour of the Microsoft Active Directory objects. value is in decimal string.
* `member_of` - The memberOf attribute of the AD object. contains object's DN.

## Import

This resource can be imported using active directory ObjectGUID of the object.

`$ terraform import activedirectory_computer.example <ObjectGUID>`

example

`$ terraform import activedirectory_computer.desktop1 e6e2b065-5a82-43bc-9fdb-6ec491de3d1d`