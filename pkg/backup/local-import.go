package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:funlen // splitting the function would add complexity
func importLocalAgents(logger *log.Logger, db database.Access, list []file.LocalAgent,
	reset bool,
) error {
	if reset {
		var servers model.LocalAgents
		if err := db.Select(&servers).Where("owner=?",
			conf.GlobalConfig.GatewayName).Run(); err != nil {
			return fmt.Errorf("failed to retrieve existing servers: %w", err)
		}

		for _, server := range servers {
			if err := db.Delete(server).Run(); err != nil {
				return fmt.Errorf("failed to delete server %q: %w", server.Name, err)
			}
		}
	}

	for i := range list {
		src := &list[i]
		// Create model with basic info to check existence
		var agent model.LocalAgent

		// Check if agent exists
		exists := true
		err := db.Get(&agent, "name=? AND owner=?", src.Name,
			conf.GlobalConfig.GatewayName).Run()

		if database.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return fmt.Errorf("failed to retrieve server %q: %w", src.Name, err)
		}

		// Populate
		agent.Name = src.Name
		agent.RootDir = src.RootDir
		agent.ReceiveDir = src.ReceiveDir
		agent.SendDir = src.SendDir
		agent.TmpReceiveDir = src.TmpReceiveDir
		agent.Address = src.Address
		agent.Protocol = src.Protocol
		agent.Disabled = src.Disabled
		agent.ProtoConfig = src.Configuration
		agent.Owner = ""

		checkLocalAgentDeprecatedFields(logger, &agent, src)

		// Create/Update
		if exists {
			logger.Info("Update local server %s\n", agent.Name)
			err = db.Update(&agent).Run()
		} else {
			logger.Info("Create local server %s\n", agent.Name)
			err = db.Insert(&agent).Run()
		}

		if err != nil {
			return fmt.Errorf("failed to import server %q: %w", agent.Name, err)
		}

		if err := importCerts(logger, db, src.Certs, &agent); err != nil {
			return err
		}

		if err := importLocalAccounts(logger, db, src.Accounts, &agent); err != nil {
			return err
		}
	}

	return nil
}

func checkLocalAgentDeprecatedFields(logger *log.Logger, agent *model.LocalAgent,
	src *file.LocalAgent,
) {
	if src.Root != "" {
		logger.Warning("JSON field 'locals.Root' is deprecated, use 'rootDir' instead")

		if agent.RootDir == "" {
			agent.RootDir = utils.DenormalizePath(src.Root)
		}
	}

	if src.InDir != "" {
		logger.Warning("JSON field 'locals.inDir' is deprecated, use 'receiveDir' instead")

		if agent.ReceiveDir == "" {
			agent.ReceiveDir = utils.DenormalizePath(src.InDir)
		}
	}

	if src.OutDir != "" {
		logger.Warning("JSON field 'locals.outDir' is deprecated, use 'sendDir' instead")

		if agent.SendDir == "" {
			agent.ReceiveDir = utils.DenormalizePath(src.OutDir)
		}
	}

	if src.WorkDir != "" {
		logger.Warning("JSON field 'locals.workDir' is deprecated, use 'tmpDir' instead")

		if agent.TmpReceiveDir == "" {
			agent.TmpReceiveDir = utils.DenormalizePath(src.WorkDir)
		}
	}
}

//nolint:dupl // duplicated code is about two different types
func importLocalAccounts(logger *log.Logger, db database.Access,
	list []file.LocalAccount, server *model.LocalAgent,
) error {
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
		}

		// Create/Update
		if exist {
			logger.Info("Update local account %s\n", account.Login)
			err = db.Update(&account).Run()
		} else {
			logger.Info("Create local account %s\n", account.Login)
			err = db.Insert(&account).Run()
		}

		if err != nil {
			return fmt.Errorf("failed to import local account %q: %w", account.Login, err)
		}

		if err := importCerts(logger, db, src.Certs, &account); err != nil {
			return err
		}
	}

	return nil
}
