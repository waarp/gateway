package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importClients(logger *log.Logger, db database.Access, clients []file.Client,
	reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.Client{}).Where("owner=?",
			conf.GlobalConfig.GatewayName).Run(); err != nil {
			return fmt.Errorf("failed to purge clients: %w", err)
		}
	}

	for _, client := range clients {
		var (
			dbClient model.Client
			isNew    bool
		)

		if err := db.Get(&dbClient, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
			client.Name).Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf("failed to retrieve client %q: %w", client.Name, err)
		}

		dbClient.Name = client.Name
		dbClient.Protocol = client.Protocol
		dbClient.Disabled = client.Disabled
		dbClient.ProtoConfig = client.ProtoConfig
		dbClient.NbOfAttempts = client.NbOfAttempts
		dbClient.FirstRetryDelay = client.FirstRetryDelay
		dbClient.RetryIncrementFactor = client.RetryIncrementFactor

		if err := dbClient.LocalAddress.Set(client.LocalAddress); err != nil {
			return fmt.Errorf("invalid client local address: %w", err)
		}

		var dbErr error

		if isNew {
			logger.Infof("Inserting new client %q", dbClient.Name)
			dbErr = db.Insert(&dbClient).Run()
		} else {
			logger.Infof("Updating existing client %q", dbClient.Name)
			dbErr = db.Update(&dbClient).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import client %q: %w", dbClient.Name, dbErr)
		}
	}

	return nil
}
