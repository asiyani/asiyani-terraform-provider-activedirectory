# Active Directory Provider

The `activedirectory` provider is used to interact with the resources supported by Microsoft Active Directory. The provider needs to be configured with the proper credentials before it can be used. Provider uses `ldap` to communicate with domain controller.

## Example Usage

```hcl
provider "activedirectory" {
  ldap_url      = "ldaps://dc1.example.com:636"
  domain        = "example.com"
  bind_username = "admin@example.com"
  bind_password = "secret_password"
}
```

## Argument Reference

The following arguments are used to configure the Active Directory Provider:

* `ldap_url` - (Required) - The LDAP URL to be used for connection. The supported schemas are: `ldap://` or `ldaps://` ie ldap://[IP]:389. it can also be sourced from the env `AD_LDAP_URL`.

* `domain` - (Required) - The AD base domain. it can also be sourced from the env `AD_DOMAIN`.

* `bind_username` - (Required) - AD service account to be used for authenticating on the AD server. it can also be sourced from the env `AD_BIND_USERNAME`.

* `bind_password` - (Required) - The password of the AD service account. it can also be sourced from the env `AD_BIND_PASSWORD`.

* `insecure_tls` - (Optional) - If true, provider skips LDAP server's SSL certificate verification (default: false). it can also be sourced from the env `AD_INSECURE_TLS`.