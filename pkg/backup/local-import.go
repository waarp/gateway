package backup

import (
	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importLocalAgents(logger *log.Logger, db *database.Session, list []LocalAgent) error {
	for _, src := range list {
		// Create model with basic info to check existence
		agent := &model.LocalAgent{
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
		agent.Root = src.Root
		agent.InDir = src.InDir
		agent.OutDir = src.OutDir
		agent.WorkDir = src.WorkDir
		agent.Address = src.Address
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration
		agent.Owner = ""

		//Create/Update
		if exists {
			logger.Infof("Update local server %s\n", agent.Name)
			err = db.Update(agent)
		} else {
			logger.Infof("Create local server %s\n", agent.Name)
			err = db.Create(agent)
		}
		if err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, "local_agents",
			agent.ID); err != nil {
			return err
		}

		if err := importLocalAccounts(logger, db, src.Accounts, agent.ID); err != nil {
			return err
		}
	}
	return nil
}

func importLocalAccounts(logger *log.Logger, db *database.Session,
	list []LocalAccount, ownerID uint64) error {

	for _, src := range list {

		// Create model with basic info to check existence
		account := &model.LocalAccount{
			LocalAgentID: ownerID,
			Login:        src.Login,
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
			logger.Infof("Update local account %s\n", account.Login)
			err = db.Update(account)
		} else {
			logger.Infof("Create local account %s\n", account.Login)
			err = db.Create(account)
		}
		if err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, "local_accounts",
			account.ID); err != nil {
			return err
		}
	}
	return nil
}
