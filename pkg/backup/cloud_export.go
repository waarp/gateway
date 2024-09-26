package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportClouds(logger *log.Logger, db database.ReadAccess) ([]file.Cloud, error) {
	var dbClouds model.CloudInstances
	if err := db.Select(&dbClouds).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud instances: %w", err)
	}

	clouds := make([]file.Cloud, len(dbClouds))

	for i, dbCloud := range dbClouds {
		clouds[i] = file.Cloud{
			Name:    dbCloud.Name,
			Type:    dbCloud.Type,
			Key:     dbCloud.Key,
			Secret:  dbCloud.Secret.String(),
			Options: dbCloud.Options,
		}

		logger.Info("Exported cloud %q", dbCloud.Name)
	}

	return clouds, nil
}
