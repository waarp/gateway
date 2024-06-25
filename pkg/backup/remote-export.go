package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

func exportRemotes(logger *log.Logger, db database.ReadAccess) ([]file.RemoteAgent, error) {
	var dbRemotes model.RemoteAgents
	if err := db.Select(&dbRemotes).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve partners: %w", err)
	}

	res := make([]file.RemoteAgent, len(dbRemotes))

	for i, src := range dbRemotes {
		accounts, err := exportRemoteAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}

		credentials, certs, _, err := exportCredentials(logger, db, src)
		if err != nil {
			return nil, err
		}

		logger.Info("Export remote partner %s", src.Name)

		res[i] = file.RemoteAgent{
			Name:          src.Name,
			Address:       src.Address.String(),
			Protocol:      src.Protocol,
			Configuration: src.ProtoConfig,
			Accounts:      accounts,
			Credentials:   credentials,
			Certs:         certs,
		}

		// Retro-compatibility with the R66 "isTLS" property.
		if src.Protocol == r66.R66TLS && compatibility.IsTLS(src.ProtoConfig) {
			res[i].Protocol = r66.R66
		}
	}

	return res, nil
}

func exportRemoteAccounts(logger *log.Logger, db database.ReadAccess,
	agentID int64,
) ([]file.RemoteAccount, error) {
	var dbAccounts model.RemoteAccounts
	if err := db.Select(&dbAccounts).Where("remote_agent_id=?", agentID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve remote accounts: %w", err)
	}

	res := make([]file.RemoteAccount, len(dbAccounts))

	for i, src := range dbAccounts {
		credentials, certs, pswd, err := exportCredentials(logger, db, src)
		if err != nil {
			return nil, err
		}

		logger.Info("Export remote account %s", src.Login)

		account := file.RemoteAccount{
			Login:       src.Login,
			Password:    pswd,
			Credentials: credentials,
			Certs:       certs,
		}
		res[i] = account
	}

	return res, nil
}
