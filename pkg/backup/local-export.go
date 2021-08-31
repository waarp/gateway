package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func exportLocals(logger *log.Logger, db database.ReadAccess) ([]file.LocalAgent, error) {
	var dbLocals model.LocalAgents
	query := db.Select(&dbLocals).Where("owner=?", conf.GlobalConfig.GatewayName)
	if err := query.Run(); err != nil {
		return nil, err
	}

	res := make([]file.LocalAgent, len(dbLocals))
	for i, src := range dbLocals {
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
			LocalInDir:    src.LocalInDir,
			LocalOutDir:   src.LocalOutDir,
			LocalTmpDir:   src.LocalTmpDir,
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
