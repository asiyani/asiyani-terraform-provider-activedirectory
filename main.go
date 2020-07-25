package main

import (
	"terraform-provider-activedirectory/activedirectory"

	"github.com/hashicorp/terraform-plugin-sdk/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: activedirectory.Provider,
	})
}
