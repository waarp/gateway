package backup

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importLocalAgents(db *database.Session, list []localAgent) error {
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
		agent.Paths = &model.ServerPaths{
			Root:    src.Root,
			InDir:   src.InDir,
			OutDir:  src.OutDir,
			WorkDir: src.WorkDir,
		}
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration
		agent.Owner = ""

		//Create/Update
		if exists {
			fmt.Printf("Update local agent %s\n", agent.Name)
			err = db.Update(agent, agent.ID, false)
		} else {
			fmt.Printf("Create local agent %s\n", agent.Name)
			err = db.Create(agent)
		}
		if err != nil {
			return err
		}

		if err := importCerts(db, src.Certs, "local_agents", agent.ID); err != nil {
			return err
		}

		if err := importLocalAccounts(db, src.Accounts, agent.ID); err != nil {
			return err
		}
	}
	return nil
}

func importLocalAccounts(db *database.Session, list []localAccount, ownerID uint64) error {

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
			fmt.Printf("Update local account %s\n", account.Login)
			err = db.Update(account, account.ID, false)
		} else {
			fmt.Printf("Create local account %s\n", account.Login)
			err = db.Create(account)
		}
		if err != nil {
			return err
		}

		if err := importCerts(db, src.Certs, "local_accounts", account.ID); err != nil {
			return err
		}
	}
	return nil
}
