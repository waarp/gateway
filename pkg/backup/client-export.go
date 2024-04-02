package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
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
		logger.Info("Exporting client %q", dbClient.Name)

		clients[i] = file.Client{
			Name:         dbClient.Name,
			Disabled:     dbClient.Disabled,
			Protocol:     dbClient.Protocol,
			LocalAddress: dbClient.LocalAddress,
			ProtoConfig:  dbClient.ProtoConfig,
		}
	}

	return clients, nil
}
