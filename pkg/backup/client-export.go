package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportClients(logger *log.Logger, db database.ReadAccess) ([]file.Client, error) {
	var dbClients model.Clients
	if err := db.Select(&dbClients).Where("owner=?",
		conf.GlobalConfig.GatewayName).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve clients: %w", err)
	}

	clients := make([]file.Client, len(dbClients))

	for i, dbClient := range dbClients {
		logger.Infof("Exporting client %q", dbClient.Name)

		clients[i] = file.Client{
			Name:                 dbClient.Name,
			Protocol:             dbClient.Protocol,
			Disabled:             dbClient.Disabled,
			LocalAddress:         dbClient.LocalAddress.String(),
			ProtoConfig:          dbClient.ProtoConfig,
			NbOfAttempts:         dbClient.NbOfAttempts,
			FirstRetryDelay:      dbClient.FirstRetryDelay,
			RetryIncrementFactor: dbClient.RetryIncrementFactor,
		}
	}

	return clients, nil
}
