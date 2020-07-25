# Terraform Provider - Active Directory

This is a Terraform Provider to work with Microsoft Active Directory. Provider supports Computer, User, Groups and OU objects, Provider also supports Computer and Users membership with Groups.

For general information about Terraform, visit the [official website][1]

[1]: https://terraform.io/

## Usage
* [install go](https://golang.org/doc/install)
* Clone Repo
* To compile provider run `make build` from root of repo. or run `go build -o terraform-provider-activedirectory`

Move binary to terraform plugins folder. [third-party-plugins](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins)

Windows	- `%APPDATA%\terraform.d\plugins`

All other systems	- `~/.terraform.d/plugins`

## Provider config
Provider needs to be configured with `Ldap URL`, `domain` name and `user credentials`. The supported schemas are: `ldap://` and `ldaps://` ie `ldap://[IP]:389`. `bind_username` should have permission to manage resources defined in tf module. [more info](./docs/index.html.markdown)

```hcl
# configure provider
provider "activedirectory" {
  ldap_url      = "ldaps://dc1.example.com:636"
  domain        = "example.com"
  bind_username = "admin@example.com"
  bind_password = "secret_password"
}
```

## Resources

### OU
`activedirectory_ou` allows you to create and configure an Active Directory OU. Arguments `name` and `base_ou_dn` are required. [more info](./docs/resources/ou.html.markdown)

```hcl
resource "activedirectory_ou" "servers" {
	name             = "servers"
	base_ou_dn       = "OU=Resources,DC=example,DC=com"
}
```

### Computer
`activedirectory_computer` allows you to create and configure an Active Directory Computer. Arguments `name`, `base_ou_dn` & `sam_account_name` are required. [more info](./docs/resources/computer.html.markdown)

```hcl
resource "activedirectory_computer" "app_server" {
  name             = "app-server"
  enabled          = true
  sam_account_name = "app-server$"
  base_ou_dn       = activedirectory_ou.servers.dn
  description      = "desktop created and maintained via terraform"
  attributes = jsonencode({
    department = ["Sales"],
    company    = ["example"]
  })
}
```

### User
`activedirectory_user` allows you to create and configure an Active Directory User. Arguments `name`, `base_ou_dn`, `sam_account_name` & `user_principal_name` are required. password is optional but if user is created without password it needs to be in disabled state. Once password is set it can not be unset it can be changed/updated. [more info](./docs/resources/user.html.markdown)

```hcl
resource "activedirectory_user" "John_Doe" {
	enabled             = true
	first_name          = "John"
	last_name           = "Doe"
	name                = "John Doe"
	sam_account_name    = "John.Doe"
	user_principal_name = "John.Doe@example.com"
	password            = "secretPassword!123"
	base_ou_dn          = "OU=Users,OU=Resources,DC=example,DC=com"
	attributes = jsonencode({
        department = ["Sales"],
        company    = ["example"],
        mail       = ["user2@example.com"],
    })
}
```

### Groups
resource `activedirectory_group` is used to manage group object. Group membership can be managed by resource `activedirectory_group_members`. [more info](./docs/resources/group.html.markdown)

```hcl
resource "activedirectory_group" "group2" {
	name             = "group2"
	sam_account_name = "group2"
	base_ou_dn       = "OU=Groups,OU=Resources,DC=example,DC=com"
	scope            = "global"
	type             = "security"
}
```


### Groups Membership
`activedirectory_group_members` is used to manage membership between group and other AD objects. This resource can be used to 
}just manage membership, Objects or group doesn't have to be managed by TF. [more info](./docs/resources/group_members.html.markdown)

```hcl
resource "activedirectory_group" "group1" {
	name             = "group1"
	sam_account_name = "group1"
	base_ou_dn       = "OU=Groups,OU=Resources,DC=example,DC=com"
}

# This resource will add server1 and server2 to group1. 
resource "activedirectory_group_members" "group1_members" {
	group_dn = activedirectory_group.group1.dn
	members  = [
        activedirectory_computer.server1.dn, 
        "cn=server2,OU=Workstations,OU=Resources,DC=example,DC=com"
        ]
}
```

### Object Membership with Groups
`activedirectory_object_memberof` can be used to add an object to multiple Groups. [more info](./docs/resources/object_memberof.html.markdown)

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