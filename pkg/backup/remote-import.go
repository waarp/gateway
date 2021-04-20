package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

func importRemoteAgents(logger *log.Logger, db database.Access, list []file.RemoteAgent) database.Error {
	for _, src := range list {
		// Create model with basic info to check existence
		var agent model.RemoteAgent

		//Check if agent exists
		exists := true
		err := db.Get(&agent, "name=?", src.Name).Run()
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

		//Create/Update
		if exists {
			logger.Infof("Update remote partner %s\n", agent.Name)
			err = db.Update(&agent).Run()
		} else {
			logger.Infof("Create remote partner %s\n", agent.Name)
			err = db.Insert(&agent).Run()
		}
		if err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, "remote_agents",
			agent.ID); err != nil {
			return err
		}

		if err := importRemoteAccounts(logger, db, src.Accounts, agent.ID); err != nil {
			return err
		}
	}
	return nil
}

//nolint:dupl
func importRemoteAccounts(logger *log.Logger, db database.Access,
	list []file.RemoteAccount, ownerID uint64) database.Error {

	for _, src := range list {

		// Create model with basic info to check existence
		var account model.RemoteAccount

		// Check if account exists
		exist, err := accountExists(db, &account, "remote_agent_id=? AND login=?",
			ownerID, src.Login)
		if err != nil {
			return err
		}

		// Populate
		account.RemoteAgentID = ownerID
		account.Login = src.Login
		if src.Password != "" {
			account.Password = types.CypherText(src.Password)
		}

		// Create/Update
		if exist {
			logger.Infof("Update remote account %s\n", account.Login)
			err = db.Update(&account).Run()
		} else {
			logger.Infof("Create remote account %s\n", account.Login)
			err = db.Insert(&account).Run()
		}
		if err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, "remote_accounts",
			account.ID); err != nil {
			return err
		}
	}
	return nil
}
