package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportRemotes(db *database.Session) ([]remoteAgent, error) {
	dbRemotes := []model.RemoteAgent{}

	if err := db.Select(&dbRemotes, nil); err != nil {
		return nil, err
	}
	res := make([]remoteAgent, len(dbRemotes))

	for i, src := range dbRemotes {

		accounts, err := exportRemoteAccounts(db, src.ID)
		if err != nil {
			return nil, err
		}
		certificates, err := exportCertificates(db, "remote_agents", src.ID)
		if err != nil {
			return nil, err
		}

		agent := remoteAgent{
			Name:          src.Name,
			Protocol:      src.Protocol,
			Configuration: src.ProtoConfig,
			Accounts:      accounts,
			Certs:         certificates,
		}
		res[i] = agent
	}
	return res, nil
}

func exportRemoteAccounts(db *database.Session, agentID uint64) ([]remoteAccount, error) {
	dbAccounts := []model.RemoteAccount{}
	filters := &database.Filters{
		Conditions: builder.Eq{"remote_agent_id": agentID},
	}
	if err := db.Select(&dbAccounts, filters); err != nil {
		return nil, err
	}
	res := make([]remoteAccount, len(dbAccounts))

	for i, src := range dbAccounts {

		certificates, err := exportCertificates(db, "remote_accounts", src.ID)
		if err != nil {
			return nil, err
		}
		pwd, err := model.DecryptPassword(src.Password)
		if err != nil {
			return nil, err
		}
		account := remoteAccount{
			Login:    src.Login,
			Password: string(pwd),
			Certs:    certificates,
		}
		res[i] = account
	}
	return res, nil
}
