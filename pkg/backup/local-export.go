package backup

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportLocals(logger *log.Logger, db database.ReadAccess) ([]file.LocalAgent, error) {
	var dbLocals model.LocalAgents
	query := db.Select(&dbLocals).Where("owner=?", database.Owner)

	if err := query.Run(); err != nil {
		return nil, err
	}

	res := make([]file.LocalAgent, len(dbLocals))

	for i := range dbLocals {
		src := &dbLocals[i]

		accounts, err := exportLocalAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}

		certificates, err := exportCertificates(logger, db, model.TableLocAgents, src.ID)
		if err != nil {
			return nil, err
		}

		logger.Infof("Export local server %s\n", src.Name)

		res[i] = file.LocalAgent{
			Name:          src.Name,
			Protocol:      src.Protocol,
			Address:       src.Address,
			Configuration: src.ProtoConfig,
			Root:          src.Root,
			LocalInDir:    src.InDir,
			LocalOutDir:   src.OutDir,
			LocalTmpDir:   src.TmpDir,
			Accounts:      accounts,
			Certs:         certificates,
		}
	}

	return res, nil
}

func exportLocalAccounts(logger *log.Logger, db database.ReadAccess,
	agentID uint64) ([]file.LocalAccount, error) {
	var dbAccounts model.LocalAccounts
	if err := db.Select(&dbAccounts).Where("local_agent_id=?", agentID).Run(); err != nil {
		return nil, err
	}

	res := make([]file.LocalAccount, len(dbAccounts))

	for i, src := range dbAccounts {
		certificates, err := exportCertificates(logger, db, model.TableLocAccounts, src.ID)
		if err != nil {
			return nil, err
		}

		logger.Infof("Export local account %s\n", src.Login)

		res[i] = file.LocalAccount{
			Login:        src.Login,
			PasswordHash: string(src.PasswordHash),
			Certs:        certificates,
		}
	}

	return res, nil
}
