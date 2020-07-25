package activedirectory

import (
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Provider for terraform activedirectory provider
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"ldap_url": {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("AD_LDAP_URL", nil),
				Description:  "The LDAP URL to be used for connection. The supported schemas are: ldap:// or  ldaps:// ie ldap://[IP]:389.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`ldap[s|i]?:\/\/`), "The supported schemas are: ldap:// or ldaps:// ie ldap://[IP]:389"),
			},
			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AD_DOMAIN", nil),
				Description: "The AD base domain.",
			},
			"bind_username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AD_BIND_USERNAME", nil),
				Description: "AD service account to be used for authenticating on the AD server.",
			},
			"bind_password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AD_BIND_PASSWORD", nil),
				Description: "The password of the AD service account.",
			},
			"insecure_tls": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AD_INSECURE_TLS", false),
				Description: "If true, skips LDAP server SSL certificate verification (default: false).",
			}},

		DataSourcesMap: map[string]*schema.Resource{
			"activedirectory_object": dataActivedirectoryObject(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"activedirectory_computer":        resourceActivedirectoryComputer(),
			"activedirectory_group":           resourceActivedirectoryGroup(),
			"activedirectory_group_members":   resourceActivedirectoryGroupMembers(),
			"activedirectory_object_memberof": resourceActivedirectoryObjectMemberOf(),
			"activedirectory_ou":              resourceActivedirectoryOU(),
			"activedirectory_user":            resourceActivedirectoryUser(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	domain := d.Get("domain").(string)

	topDN := "dc=" + strings.Replace(domain, ".", ",dc=", -1)

	client := &ADClient{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:       hclog.LevelFromString(os.Getenv("TF_LOG")),
			DisableTime: true, //timestamp is provided by TF Debug logs
		}),
		config: Config{
			serverURL:   d.Get("ldap_url").(string),
			domain:      strings.ToLower(domain),
			topDN:       strings.ToLower(topDN),
			username:    d.Get("bind_username").(string),
			password:    d.Get("bind_password").(string),
			insecureTLS: d.Get("insecure_tls").(bool),
		},
	}
	client.logger.Debug("providerConfigure: ad client initialised")
	return client, nil
}
