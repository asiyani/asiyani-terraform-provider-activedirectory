# activedirectory_object

Use this data source to retrieve `computer`, `user`, `group` or `OU` from Active Directory. 

## Example Usage

```hcl
data "activedirectory_object" "desktop1" {
    dn = "CN=desktop1,OU=Workstations,OU=Resources,DC=example,DC=com"
}

data "activedirectory_object" "user" {
    guid = "0f794fec-e4f5-4589-8618-941e2f66419a"
}
```

## Argument Reference

* `dn` - (Optional) - The distinguished name (dn) of the object.
* `guid` - (Optional) - The `ObjectGUID` of the object. value is in hexadecimal format and in Endian Ordering used by Microsoft Active Directory.

## Attribute Reference

* `name` - The name of the Object.
* `cn` - The Common-Name property of the object.
* `user_principal_name` - The userPrincipalName for user object.
* `sam_account_name` - The sAMAccountName attribute of the object.
* `base_ou_dn` - The `dn` (distinguished name) of the base ou of the object.
* `members` - The member attribute of the AD object. contains object's DN.
* `member_of` - The memberOf attribute of the AD object. contains object's DN.
* `description` - A description for the AD object.
* `sid` - The security identifier (SID) of the object.