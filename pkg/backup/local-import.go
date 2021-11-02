package backup

import (
	"fmt"

	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:funlen // splitting the function would add complexity
func importLocalAgents(logger *log.Logger, db database.Access, list []file.LocalAgent) database.Error {
	for i := range list {
		src := &list[i]
		// Create model with basic info to check existence
		var agent model.LocalAgent

		// Check if agent exists
		exists := true
		err := db.Get(&agent, "name=? AND owner=?", src.Name, database.Owner).Run()

		if database.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		}

		// Populate
		agent.Name = src.Name
		agent.Root = src.Root
		agent.InDir = src.InDir
		agent.OutDir = src.OutDir
		agent.TmpDir = src.TmpDir
		agent.Address = src.Address
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration
		agent.Owner = ""

		if agent.TmpDir == "" && src.WorkDir != "" {
			logger.Warning("JSON field 'locals.workDir' is deprecated, use 'tmpDir' instead")

			agent.TmpDir = src.WorkDir
		}

		// Create/Update
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

		if err := importCerts(logger, db, src.Certs, model.TableLocAgents,
			agent.ID); err != nil {
			return err
		}

		if err := importLocalAccounts(logger, db, src.Accounts, &agent); err != nil {
			return err
		}
	}

	return nil
}

//nolint:dupl // duplicated code is about two different types
func importLocalAccounts(logger *log.Logger, db database.Access,
	list []file.LocalAccount, server *model.LocalAgent) database.Error {
	for _, src := range list {
		// Create model with basic info to check existence
		var account model.LocalAccount

		// Check if account exists
		exist, err := accountExists(db, &account, "local_agent_id=? AND login=?",
			server.ID, src.Login)
		if err != nil {
			return err
		}

		// Populate
		account.LocalAgentID = server.ID
		account.Login = src.Login

		if src.PasswordHash != "" {
			account.PasswordHash = src.PasswordHash
		} else if src.Password != "" {
			pswd := src.Password
			if server.Protocol == "r66" {
				// Unlike other protocols, when authenticating, an R66 client sends a
				// hash instead of a password, so we replace the password with its hash.
				pswd = string(r66.CryptPass([]byte(pswd)))
			}
			var err error
			if account.PasswordHash, err = utils.HashPassword(database.BcryptRounds, pswd); err != nil {
				return database.NewInternalError(fmt.Errorf("failed to hash account password: %w", err))
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

		if err := importCerts(logger, db, src.Certs, model.TableLocAccounts,
			account.ID); err != nil {
			return err
		}
	}

	return nil
}
