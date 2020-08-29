package activedirectory

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-ldap/ldap/v3"
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
				Description: "The AD domain.",
			},
			"top_dn": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AD_TOP_DN", nil),
				Description: "The AD base domain to use.",
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
	topDN := d.Get("top_dn").(string)
	domainDN := strings.ToLower("dc=" + strings.Replace(domain, ".", ",dc=", -1))

	if topDN == "" {
		topDN = domainDN
	}

	if err := validateTopDNString(domainDN, topDN); err != nil {
		return nil, fmt.Errorf("unable to verify top_dn err:%w", err)
	}

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

func validateTopDNString(domainDN, topDN string) error {
	var errStr string
	if !strings.HasSuffix(strings.ToLower(topDN), domainDN) {
		errStr += fmt.Sprintf(`top_dn should end with domain component %q : `, domainDN)
	}
	if _, err := ldap.ParseDN(strings.ToLower(topDN)); err != nil {
		errStr += fmt.Sprintf("top_dn is not a valid DN err: %v", err)
	}
	if errStr == "" {
		return nil
	}
	return fmt.Errorf("error: %s, got: %s", errStr, topDN)
}
