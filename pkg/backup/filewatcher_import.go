package backup

import (
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importFilewatchers(logger *log.Logger, db database.Access, config *file.FileWatchers,
	reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.FileWatcher{}).Run(); err != nil {
			return fmt.Errorf("failed to delete existing remote filewatchers: %w", err)
		}
	}

	if config == nil {
		return nil
	}

	return importRemoteFilewatchers(logger, db, config.Remote)
}

func importRemoteFilewatchers(logger *log.Logger, db database.Access, fws []file.RemoteFilewatcher,
) error {
	for _, filewatcher := range fws {
		var dbFW model.FileWatcher
		if err := db.Get(&dbFW, "flow=?", filewatcher.Flow).Run(); err != nil &&
			!database.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve remote filewatcher %q: %w", filewatcher.Flow, err)
		}

		var rule model.Rule
		if err := db.Get(&rule, "name=?", filewatcher.Rule).And("is_send=?", false).Run(); err != nil {
			return fmt.Errorf("failed to retrieve rule %q: %w", filewatcher.Rule, err)
		}

		var client model.Client
		if err := db.Get(&client, "name=?", filewatcher.Client).Run(); err != nil {
			return fmt.Errorf("failed to retrieve client %q: %w", filewatcher.Client, err)
		}

		var partner model.RemoteAgent
		if err := db.Get(&partner, "name=?", filewatcher.Partner).Run(); err != nil {
			return fmt.Errorf("failed to retrieve partner %q: %w", filewatcher.Partner, err)
		}

		var account model.RemoteAccount
		if err := db.Get(&account, "login=?", filewatcher.RemoteAccount).
			And("remote_agent_id=?", partner.ID).Run(); err != nil {
			return fmt.Errorf("failed to retrieve account %q for partner %q: %w",
				filewatcher.RemoteAccount, filewatcher.Partner, err)
		}

		dbFW.Flow = filewatcher.Flow
		dbFW.Interval = time.Duration(filewatcher.Interval)
		dbFW.Pattern = filewatcher.Pattern
		dbFW.RemoteAccount = account
		dbFW.Client = client
		dbFW.Rule = rule
		dbFW.NoDuplicateCheck = filewatcher.NoDuplicateCheck

		var dbErr error
		if dbFW.ID == 0 {
			logger.Infof("Inserting new remote filewatcher %q", dbFW.Flow)
			dbErr = db.Insert(&dbFW).Run()
		} else {
			logger.Infof("Updating existing remote filewatcher %q", dbFW.Flow)
			dbErr = db.Update(&dbFW).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import remote filewatcher %q: %w", dbFW.Flow, dbErr)
		}
	}

	return nil
}
