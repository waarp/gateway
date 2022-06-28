package backup

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportRemotes(logger *log.Logger, db database.ReadAccess) ([]file.RemoteAgent, error) {
	var dbRemotes model.RemoteAgents
	if err := db.Select(&dbRemotes).Run(); err != nil {
		return nil, err
	}

	res := make([]file.RemoteAgent, len(dbRemotes))

	for i, src := range dbRemotes {
		accounts, err := exportRemoteAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}

		certificates, err := exportCertificates(logger, db, model.TableRemAgents, src.ID)
		if err != nil {
			return nil, err
		}

		logger.Info("Export remote partner %s\n", src.Name)

		agent := file.RemoteAgent{
			Name:          src.Name,
			Protocol:      src.Protocol,
			Address:       src.Address,
			Configuration: src.ProtoConfig,
			Accounts:      accounts,
			Certs:         certificates,
		}
		res[i] = agent
	}

	return res, nil
}

func exportRemoteAccounts(logger *log.Logger, db database.ReadAccess,
	agentID uint64,
) ([]file.RemoteAccount, error) {
	var dbAccounts model.RemoteAccounts
	if err := db.Select(&dbAccounts).Where("remote_agent_id=?", agentID).Run(); err != nil {
		return nil, err
	}

	res := make([]file.RemoteAccount, len(dbAccounts))

	for i, src := range dbAccounts {
		certificates, err := exportCertificates(logger, db, model.TableRemAccounts, src.ID)
		if err != nil {
			return nil, err
		}

		logger.Info("Export remote account %s\n", src.Login)

		account := file.RemoteAccount{
			Login:    src.Login,
			Password: string(src.Password),
			Certs:    certificates,
		}
		res[i] = account
	}

	return res, nil
}
