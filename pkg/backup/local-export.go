package backup

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func exportLocals(logger *log.Logger, db database.ReadAccess) ([]file.LocalAgent, error) {
	var dbLocals model.LocalAgents
	if err := db.Select(&dbLocals).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return nil, err
	}

	res := make([]file.LocalAgent, len(dbLocals))

	for i, src := range dbLocals {
		accounts, err := exportLocalAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}

		certificates, err := exportCertificates(logger, db, src)
		if err != nil {
			return nil, err
		}

		logger.Info("Export local server %s\n", src.Name)

		res[i] = file.LocalAgent{
			Name:          src.Name,
			Protocol:      src.Protocol,
			Address:       src.Address,
			Configuration: src.ProtoConfig,
			RootDir:       src.RootDir,
			ReceiveDir:    src.ReceiveDir,
			SendDir:       src.SendDir,
			TmpReceiveDir: src.TmpReceiveDir,
			Root:          utils.NormalizePath(src.RootDir),
			InDir:         utils.NormalizePath(src.ReceiveDir),
			OutDir:        utils.NormalizePath(src.SendDir),
			WorkDir:       utils.NormalizePath(src.TmpReceiveDir),
			Accounts:      accounts,
			Certs:         certificates,
		}
	}

	return res, nil
}

func exportLocalAccounts(logger *log.Logger, db database.ReadAccess,
	agentID int64,
) ([]file.LocalAccount, error) {
	var dbAccounts model.LocalAccounts
	if err := db.Select(&dbAccounts).Where("local_agent_id=?", agentID).Run(); err != nil {
		return nil, err
	}

	res := make([]file.LocalAccount, len(dbAccounts))

	for i, src := range dbAccounts {
		certificates, err := exportCertificates(logger, db, src)
		if err != nil {
			return nil, err
		}

		logger.Info("Export local account %s\n", src.Login)

		res[i] = file.LocalAccount{
			Login:        src.Login,
			PasswordHash: src.PasswordHash,
			Certs:        certificates,
		}
	}

	return res, nil
}
