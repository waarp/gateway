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

func exportLocals(logger *log.Logger, db database.ReadAccess) ([]file.LocalAgent, error) {
	var dbLocals model.LocalAgents
	if err := db.Select(&dbLocals).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve partners: %w", err)
	}

	res := make([]file.LocalAgent, len(dbLocals))

	for i, src := range dbLocals {
		accounts, err := exportLocalAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}

		credentials, certs, _, err := exportCredentials(logger, db, src)
		if err != nil {
			return nil, err
		}

		logger.Info("Export local server %s", src.Name)

		res[i] = file.LocalAgent{
			Name:          src.Name,
			Protocol:      src.Protocol,
			Disabled:      src.Disabled,
			Address:       src.Address.String(),
			Configuration: src.ProtoConfig,
			RootDir:       src.RootDir,
			ReceiveDir:    src.ReceiveDir,
			SendDir:       src.SendDir,
			TmpReceiveDir: src.TmpReceiveDir,
			Accounts:      accounts,
			Credentials:   credentials,
			Root:          utils.NormalizePath(src.RootDir),
			InDir:         utils.NormalizePath(src.ReceiveDir),
			OutDir:        utils.NormalizePath(src.SendDir),
			WorkDir:       utils.NormalizePath(src.TmpReceiveDir),
			Certs:         certs,
		}
	}

	return res, nil
}

func exportLocalAccounts(logger *log.Logger, db database.ReadAccess,
	agentID int64,
) ([]file.LocalAccount, error) {
	var dbAccounts model.LocalAccounts
	if err := db.Select(&dbAccounts).Where("local_agent_id=?", agentID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve local accounts: %w", err)
	}

	res := make([]file.LocalAccount, len(dbAccounts))

	for i, src := range dbAccounts {
		credentials, certs, pswd, err := exportCredentials(logger, db, src)
		if err != nil {
			return nil, err
		}

		logger.Info("Export local account %s", src.Login)

		res[i] = file.LocalAccount{
			Login:        src.Login,
			Credentials:  credentials,
			PasswordHash: pswd,
			Certs:        certs,
		}
	}

	return res, nil
}
