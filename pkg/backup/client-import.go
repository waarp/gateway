package backup

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importClients(logger *log.Logger, db database.Access, clients []file.Client,
	reset bool,
) database.Error {
	if reset {
		if err := db.DeleteAll(&model.Client{}).Where("owner=?",
			conf.GlobalConfig.GatewayName).Run(); err != nil {
			return err
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
			return err
		}

		dbClient.Name = client.Name
		dbClient.Protocol = client.Protocol
		dbClient.Disabled = client.Disabled
		dbClient.LocalAddress = client.LocalAddress
		dbClient.ProtoConfig = client.ProtoConfig

		var dbErr database.Error

		if isNew {
			logger.Info("Inserting new client %q", dbClient.Name)
			dbErr = db.Insert(&dbClient).Run()
		} else {
			logger.Info("Updating existing client %q", dbClient.Name)
			dbErr = db.Update(&dbClient).Run()
		}

		if dbErr != nil {
			return dbErr
		}
	}

	return nil
}
