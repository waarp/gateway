package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportLocals(logger *log.Logger, db *database.Session) ([]localAgent, error) {
	dbLocals := []model.LocalAgent{}
	filter := database.Filters{
		Conditions: builder.Eq{"owner": database.Owner},
	}

	if err := db.Select(&dbLocals, &filter); err != nil {
		return nil, err
	}

	res := make([]localAgent, len(dbLocals))

	for i, src := range dbLocals {
		accounts, err := exportLocalAccounts(logger, db, src.ID)
		if err != nil {
			return nil, err
		}

		certificates, err := exportCertificates(logger, db, "local_agents", src.ID)
		if err != nil {
			return nil, err
		}

		logger.Infof("Export local server %s\n", src.Name)
		res[i] = localAgent{
			Name:          src.Name,
			Protocol:      src.Protocol,
			Configuration: src.ProtoConfig,
			Root:          src.Paths.Root,
			InDir:         src.Paths.InDir,
			OutDir:        src.Paths.OutDir,
			WorkDir:       src.Paths.WorkDir,
			Accounts:      accounts,
			Certs:         certificates,
		}
	}

	return res, nil
}

func exportLocalAccounts(logger *log.Logger, db *database.Session,
	agentID uint64) ([]localAccount, error) {

	dbAccounts := []model.LocalAccount{}
	filters := &database.Filters{
		Conditions: builder.Eq{"local_agent_id": agentID},
	}

	if err := db.Select(&dbAccounts, filters); err != nil {
		return nil, err
	}

	res := make([]localAccount, len(dbAccounts))

	for i, src := range dbAccounts {

		certificates, err := exportCertificates(logger, db, "local_accounts", src.ID)
		if err != nil {
			return nil, err
		}

		logger.Infof("Export local account %s\n", src.Login)
		res[i] = localAccount{
			Login:    src.Login,
			Password: string(src.Password),
			Certs:    certificates,
		}
	}

	return res, nil
}
