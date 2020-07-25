package activedirectory

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sharedClient() (interface{}, error) {
	if v := os.Getenv("AD_LDAP_URL"); v == "" {
		return nil, fmt.Errorf("AD_LDAP_URL must be set for sweep tests")
	}
	if v := os.Getenv("AD_DOMAIN"); v == "" {
		return nil, fmt.Errorf("AD_DOMAIN must be set for sweep tests")
	}
	if v := os.Getenv("AD_BIND_USERNAME"); v == "" {
		return nil, fmt.Errorf("AD_BIND_USERNAME must be set for sweep tests")
	}
	if v := os.Getenv("AD_BIND_PASSWORD"); v == "" {
		return nil, fmt.Errorf("AD_BIND_PASSWORD must be set for sweep tests")
	}
	var insecureTLS bool
	if strings.EqualFold(os.Getenv("AD_INSECURE_TLS"), "true") {
		insecureTLS = true
	}

	domain := os.Getenv("AD_DOMAIN")

	topDN := "dc=" + strings.Replace(domain, ".", ",dc=", -1)

	client := &ADClient{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:       hclog.LevelFromString(os.Getenv("TF_LOG")),
			DisableTime: true,
		}),
		config: Config{
			serverURL:   os.Getenv("AD_LDAP_URL"),
			domain:      strings.ToLower(domain),
			topDN:       strings.ToLower(topDN),
			username:    os.Getenv("AD_BIND_USERNAME"),
			password:    os.Getenv("AD_BIND_PASSWORD"),
			insecureTLS: insecureTLS,
		},
	}
	return client, nil
}
