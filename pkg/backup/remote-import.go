package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:funlen //splitting would add complexity
func importRemoteAgents(logger *log.Logger, db database.Access,
	list []file.RemoteAgent, reset bool,
) error {
	if reset {
		var partners model.RemoteAgents
		if err := db.Select(&partners).Run(); err != nil {
			return fmt.Errorf("failed to retrieve existing partners: %w", err)
		}

		for _, partner := range partners {
			if err := db.Delete(partner).Run(); err != nil {
				return fmt.Errorf("failed to delete partner %q: %w", partner.Name, err)
			}
		}
	}

	for i := range list {
		src := &list[i]

		// Create model with basic info to check existence
		var agent model.RemoteAgent

		// Check if agent exists
		exists := true
		if err := db.Get(&agent, "name=? AND owner=?", src.Name,
			conf.GlobalConfig.GatewayName).Run(); database.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return fmt.Errorf("failed to retrieve partner %q: %w", src.Name, err)
		}

		// Populate
		agent.Name = src.Name
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration

		if err := agent.Address.Set(src.Address); err != nil {
			return database.NewValidationError(err.Error())
		}

		var dbErr error

		// Create/Update
		if exists {
			logger.Infof("Update remote partner %q", agent.Name)
			dbErr = db.Update(&agent).Run()
		} else {
			logger.Infof("Create remote partner %q", agent.Name)
			dbErr = db.Insert(&agent).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to create/update partner %q: %w", agent.Name, dbErr)
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
) error {
	for _, src := range list {
		// Create model with basic info to check existence
		var account model.RemoteAccount

		// Check if account exists
		exist, dbErr := accountExists(db, &account, "remote_agent_id=? AND login=?",
			partner.ID, src.Login)
		if dbErr != nil {
			return dbErr
		}

		// Populate
		account.RemoteAgentID = partner.ID
		account.Login = src.Login

		// Create/Update
		if exist {
			logger.Infof("Update remote account %q", account.Login)
			dbErr = db.Update(&account).Run()
		} else {
			logger.Infof("Create remote account %q", account.Login)
			dbErr = db.Insert(&account).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to create/update remote account %q: %w", account.Login, dbErr)
		}

		if src.Password != "" {
			pswd := &model.Credential{
				RemoteAccountID: utils.NewNullInt64(account.ID),
				Type:            auth.Password,
				Value:           src.Password,
			}

			if err := db.DeleteAll(&model.Credential{}).Where("type=?", pswd.Type).
				Where(account.GetCredCond()).Run(); err != nil {
				return fmt.Errorf("failed to delete the old password: %w", err)
			}

			if err := db.Insert(pswd).Run(); err != nil {
				return fmt.Errorf("failed to insert the new password: %w", err)
			}
		}

		if err := credentialsImport(logger, db, src.Credentials, &account); err != nil {
			return err
		}

		if err := importCerts(logger, db, src.Certs, &account); err != nil {
			return err
		}
	}

	return nil
}
