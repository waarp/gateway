package backup

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importRemoteAgents(db *database.Session, list []remoteAgent) error {
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
		agent.Protocol = src.Protocol
		agent.ProtoConfig = src.Configuration

		//Create/Update
		if exists {
			fmt.Printf("Update remote agent %s\n", agent.Name)
			err = db.Update(agent, agent.ID, false)
		} else {
			fmt.Printf("Create remote agent %s\n", agent.Name)
			err = db.Create(agent)
		}
		if err != nil {
			return err
		}

		if err := importCerts(db, src.Certs, "remote_agents", agent.ID); err != nil {
			return err
		}

		if err := importRemoteAccounts(db, src.Accounts, agent.ID); err != nil {
			return err
		}
	}
	return nil
}

func importRemoteAccounts(db *database.Session, list []remoteAccount, ownerID uint64) error {

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
			fmt.Printf("Update remote account %s\n", account.Login)
			err = db.Update(account, account.ID, false)
		} else {
			fmt.Printf("Create remote account %s\n", account.Login)
			err = db.Create(account)
		}
		if err != nil {
			return err
		}

		if err := importCerts(db, src.Certs, "remote_accounts", account.ID); err != nil {
			return err
		}
	}
	return nil
}
