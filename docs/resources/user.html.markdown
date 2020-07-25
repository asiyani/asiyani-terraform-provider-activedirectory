# activedirectory_user

This resource allows you to create and configure an Active Directory User.

## Example Usage

```hcl
# basic example
resource "activedirectory_user" "user1" {
    enabled             = false
    name                = "user1"
    user_principal_name = "user1@example.com"
    sam_account_name    = "user1"
    base_ou_dn          = "OU=Users,OU=Resources,DC=example,DC=com"
}

resource "activedirectory_user" "John_Doe" {
	enabled             = true
	first_name          = "John"
	last_name           = "Doe"
	name                = "John Doe"
	sam_account_name    = "John.Doe"
	user_principal_name = "John.Doe@example.com"
	password            = "secretPassword!123"
	base_ou_dn          = "OU=Users,OU=Resources,DC=example,DC=com"
	description         = "user created by terraform"
	attributes = jsonencode({
        department = ["Sales"],
        company    = ["example"],
        mail       = ["user2@example.com"],
    })
}
```

## Argument Reference

* `name` - (Required) - The name of the Object.
* `base_ou_dn` - (Required) - The `dn` (distinguished name) of the `OU` (Organizational Unit) or container where the object is created.
* `sam_account_name` - (Required) - The sAMAccountName attribute of the object. It must be 20 or fewer characters.
* `user_principal_name` - (Required) - The userPrincipalName for user object. should be in format `someone@domain.com`.

* `first_name` - (Optional) - A firstname/givenname of the user object.
* `last_name` - (Optional) - A lastname/sn of the user object.
* `password`- (Optional) - The password for user object. password is optional but if user is created without password it needs to be in disabled state. Once password is set it can not be unset it can be changed/updated.
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

`$ terraform import activedirectory_user.example <ObjectGUID>`

example

`$ terraform import activedirectory_user.user1 e6e2b065-5a82-43bc-9fdb-6ec491de3d1d`