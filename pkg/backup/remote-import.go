package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importRemoteAgents(logger *log.Logger, db *database.Session, list []remoteAgent) error {
	for _, src := range list {
		// Create model with basic info to check existence
		agent := &model.RemoteAgent{
			Name: src.Name,
		}

		//Check if agent exists
		exists := true
		err := db.Get(agent)
		if err != nil {
			if err == database.ErrNotFound {
				exists = false
			} else {
				return err
			}
		}

		// Populate
		agent.Address = src.Address
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration

		//Create/Update
		if exists {
			logger.Infof("Update remote partner %s\n", agent.Name)
			err = db.Update(agent)
		} else {
			logger.Infof("Create remote partner %s\n", agent.Name)
			err = db.Create(agent)
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

func importRemoteAccounts(logger *log.Logger, db *database.Session,
	list []remoteAccount, ownerID uint64) error {

	for _, src := range list {

		// Create model with basic info to check existence
		account := &model.RemoteAccount{
			RemoteAgentID: ownerID,
			Login:         src.Login,
		}

		// Check if account exists
		exist, err := accountExists(db, account)
		if err != nil {
			return err
		}

		// Populate
		if src.Password != "" {
			account.Password = []byte(src.Password)
		}

		// Create/Update
		if exist {
			logger.Infof("Update remote account %s\n", account.Login)
			err = db.Update(account)
		} else {
			logger.Infof("Create remote account %s\n", account.Login)
			err = db.Create(account)
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
