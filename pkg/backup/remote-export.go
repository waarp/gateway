package backup

import (
	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportRemotes(logger *log.Logger, db *database.Session) ([]RemoteAgent, error) {
	var dbRemotes []model.RemoteAgent

	if err := db.Select(&dbRemotes, nil); err != nil {
		return nil, err
	}
	res := make([]RemoteAgent, len(dbRemotes))

	for i, src := range dbRemotes {

		accounts, err := exportRemoteAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}
		certificates, err := exportCertificates(logger, db, "remote_agents", src.ID)
		if err != nil {
			return nil, err
		}

		logger.Infof("Export remote partner %s\n", src.Name)
		agent := RemoteAgent{
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

func exportRemoteAccounts(logger *log.Logger, db *database.Session,
	agentID uint64) ([]RemoteAccount, error) {
	var dbAccounts []model.RemoteAccount
	filters := &database.Filters{
		Conditions: builder.Eq{"remote_agent_id": agentID},
	}
	if err := db.Select(&dbAccounts, filters); err != nil {
		return nil, err
	}
	res := make([]RemoteAccount, len(dbAccounts))

	for i, src := range dbAccounts {

		certificates, err := exportCertificates(logger, db, "remote_accounts", src.ID)
		if err != nil {
			return nil, err
		}
		pwd, err := model.DecryptPassword(src.Password)
		if err != nil {
			return nil, err
		}

		logger.Infof("Export remote account %s\n", src.Login)
		account := RemoteAccount{
			Login:    src.Login,
			Password: string(pwd),
			Certs:    certificates,
		}
		res[i] = account
	}
	return res, nil
}
