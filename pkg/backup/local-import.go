package backup

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func importLocalAgents(logger *log.Logger, db database.Access, list []file.LocalAgent) database.Error {
	for _, src := range list {
		// Create model with basic info to check existence
		var agent model.LocalAgent

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
		agent.Root = src.Root
		agent.LocalInDir = src.LocalInDir
		agent.LocalOutDir = src.LocalOutDir
		agent.LocalTmpDir = src.LocalTmpDir
		agent.Address = src.Address
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration
		agent.Owner = ""
		if src.InDir != "" {
			logger.Warning("JSON field 'locals.inDir' is deprecated, use 'localInDir' instead")
			agent.LocalInDir = src.InDir
		}
		if src.OutDir != "" {
			logger.Warning("JSON field 'locals.outDir' is deprecated, use 'localOutDir' instead")
			agent.LocalOutDir = src.OutDir
		}
		if src.WorkDir != "" {
			logger.Warning("JSON field 'locals.workDir' is deprecated, use 'localTmpDir' instead")
			agent.LocalTmpDir = src.WorkDir
		}

		//Create/Update
		if exists {
			logger.Infof("Update local server %s\n", agent.Name)
			err = db.Update(&agent).Run()
		} else {
			logger.Infof("Create local server %s\n", agent.Name)
			err = db.Insert(&agent).Run()
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

//nolint:dupl
func importLocalAccounts(logger *log.Logger, db database.Access,
	list []file.LocalAccount, ownerID uint64) database.Error {

	for _, src := range list {

		// Create model with basic info to check existence
		var account model.LocalAccount

		// Check if account exists
		exist, err := accountExists(db, &account, "local_agent_id=? AND login=?",
			ownerID, src.Login)
		if err != nil {
			return err
		}

		// Populate
		account.LocalAgentID = ownerID
		account.Login = src.Login
		if src.PasswordHash != "" {
			account.PasswordHash = []byte(src.PasswordHash)
		} else if src.Password != "" {
			var err error
			if account.PasswordHash, err = utils.HashPassword(database.BcryptRounds,
				[]byte(src.Password)); err != nil {
				return database.NewInternalError(fmt.Errorf("failed to hash account password: %s", err))
			}

		}

		// Create/Update
		if exist {
			logger.Infof("Update local account %s\n", account.Login)
			err = db.Update(&account).Run()
		} else {
			logger.Infof("Create local account %s\n", account.Login)
			err = db.Insert(&account).Run()
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
