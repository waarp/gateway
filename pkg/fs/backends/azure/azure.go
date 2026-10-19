package azure

import (
	"github.com/rclone/rclone/fs/config/configmap"
)

func azureCredOpts(account, key string) configmap.Simple {
	return configmap.Simple{
		"account": account,
		"key":     key,
	}
}
