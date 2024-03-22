package backup

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

//nolint:funlen //splitting would add complexity
func importRemoteAgents(logger *log.Logger, db database.Access,
	list []file.RemoteAgent, reset bool,
) database.Error {
	if reset {
		var partners model.RemoteAgents
		if err := db.Select(&partners).Run(); err != nil {
			return err
		}

		for _, partner := range partners {
			if err := db.Delete(partner).Run(); err != nil {
				return err
			}
		}
	}

	for _, src := range list {
		// Create model with basic info to check existence
		var agent model.RemoteAgent

		// Check if agent exists
		exists := true
		err := db.Get(&agent, "name=? AND owner=?", src.Name,
			conf.GlobalConfig.GatewayName).Run()

		if database.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		}

		// Populate
		agent.Name = src.Name
		agent.Address = src.Address
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration

		// Create/Update
		if exists {
			logger.Info("Update remote partner %s\n", agent.Name)
			err = db.Update(&agent).Run()
		} else {
			logger.Info("Create remote partner %s\n", agent.Name)
			err = db.Insert(&agent).Run()
		}

		if err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, &agent); err != nil {
			return err
		}

		if err := importRemoteAccounts(logger, db, src.Accounts, &agent); err != nil {
			return err
		}
	}

	return nil
}

//nolint:dupl // duplicated code is about two different types
func importRemoteAccounts(logger *log.Logger, db database.Access,
	list []file.RemoteAccount, partner *model.RemoteAgent,
) database.Error {
	for _, src := range list {
		// Create model with basic info to check existence
		var account model.RemoteAccount

		// Check if account exists
		exist, err := accountExists(db, &account, "remote_agent_id=? AND login=?",
			partner.ID, src.Login)
		if err != nil {
			return err
		}

		// Populate
		account.RemoteAgentID = partner.ID
		account.Login = src.Login

		if src.Password != "" {
			account.Password = types.CypherText(src.Password)
		}

		// Create/Update
		if exist {
			logger.Info("Update remote account %s\n", account.Login)
			err = db.Update(&account).Run()
		} else {
			logger.Info("Create remote account %s\n", account.Login)
			err = db.Insert(&account).Run()
		}

		if err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, &account); err != nil {
			return err
		}
	}

	return nil
}
