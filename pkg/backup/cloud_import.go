package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importCloud(logger *log.Logger, db database.Access, clouds []file.Cloud, reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.CloudInstance{}).Run(); err != nil {
			return fmt.Errorf("failed to purge existing cloud instances: %w", err)
		}
	}

	for _, cloud := range clouds {
		exists := false

		var dbCloud model.CloudInstance
		if err := db.Get(&dbCloud, "name=?", cloud.Name,
			conf.GlobalConfig.GatewayName).Run(); database.IsNotFound(err) {
			exists = true
		} else if err != nil {
			return fmt.Errorf("failed to retrieve cloud instance %q: %w", cloud.Name, err)
		}

		dbCloud.Name = cloud.Name
		dbCloud.Type = cloud.Type
		dbCloud.Key = cloud.Key
		dbCloud.Secret = database.SecretText(cloud.Secret)
		dbCloud.Options = cloud.Options

		var (
			dbErr  error
			logMsg string
		)

		if exists {
			dbErr = db.Insert(&dbCloud).Run()
			logMsg = fmt.Sprintf("Inserted new cloud instance %q", cloud.Name)
		} else {
			dbErr = db.Update(&dbCloud).Run()
			logMsg = fmt.Sprintf("Updated existing cloud instance %q", cloud.Name)
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import cloud instance %q: %w", cloud.Name, dbErr)
		}

		logger.Info(logMsg)
	}

	return nil
}
