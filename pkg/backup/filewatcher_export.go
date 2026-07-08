package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportFilewatchers(logger *log.Logger, db database.ReadAccess,
) (*file.FileWatchers, error) {
	remoteFW, err := exportRemoteFilewatchers(logger, db)
	if err != nil {
		return nil, err
	}

	return &file.FileWatchers{
		Remote: remoteFW,
	}, nil
}

func exportRemoteFilewatchers(logger *log.Logger, db database.ReadAccess,
) ([]file.RemoteFilewatcher, error) {
	var dbFWs model.FileWatchers
	if err := db.Select(&dbFWs).Eager().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve remote filewatchers: %w", err)
	}

	res := make([]file.RemoteFilewatcher, len(dbFWs))

	for i, dbFW := range dbFWs {
		var partner model.RemoteAgent
		if err := db.Get(&partner, "id=?", dbFW.RemoteAccount.RemoteAgentID).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve partner for filewatcher %q: %w", dbFW.Flow, err)
		}

		res[i] = file.RemoteFilewatcher{
			Flow:             dbFW.Flow,
			Interval:         file.Duration(dbFW.Interval),
			Pattern:          dbFW.Pattern,
			Partner:          partner.Name,
			RemoteAccount:    dbFW.RemoteAccount.Login,
			Client:           dbFW.Client.Name,
			Rule:             dbFW.Rule.Name,
			NoDuplicateCheck: dbFW.NoDuplicateCheck,
		}

		logger.Infof("Exported filewatcher %q", dbFW.Flow)
	}

	return res, nil
}
