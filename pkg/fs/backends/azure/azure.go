package azure

import (
	"maps"

	"github.com/rclone/rclone/fs/config/configmap"
)

func parseAzureOpts(account, key string, opts map[string]string) configmap.Simple {
	const (
		optsAccountKey = "account"
		optsKeyKey     = "key"
	)

	config := maps.Clone(opts)

	if account != "" {
		config[optsAccountKey] = account
	}

	if key != "" {
		config[optsKeyKey] = key
	}

	return config
}
