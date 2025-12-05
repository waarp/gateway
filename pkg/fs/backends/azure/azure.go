package azure

import (
	"github.com/rclone/rclone/fs/config/configmap"
)

func parseAzureOpts(account, key string, opts map[string]string) configmap.Simple {
	const (
		optsAccountKey = "account"
		optsKeyKey     = "key"
	)

	if account != "" {
		opts[optsAccountKey] = account
	}

	if key != "" {
		opts[optsKeyKey] = key
	}

	return opts
}
