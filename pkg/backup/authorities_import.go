package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importAuthorities(logger *log.Logger, db database.Access,
	authorities []*file.Authority, reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.Authority{}).Run(); err != nil {
			return fmt.Errorf("failed to reset authority table: %w", err)
		}
	}

	for _, authority := range authorities {
		var dbAuth model.Authority
		if err := db.Get(&dbAuth, "name=?", authority.Name).Run(); err != nil &&
			!database.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve authority %q: %w", authority.Name, err)
		}

		dbAuth.Name = authority.Name
		dbAuth.Type = authority.Type
		dbAuth.PublicIdentity = authority.PublicIdentity
		dbAuth.ValidHosts = authority.ValidHosts

		var dbErr error

		if dbAuth.ID == 0 {
			logger.Info("Inserting new authority %q", dbAuth.Name)
			dbErr = db.Insert(&dbAuth).Run()
		} else {
			logger.Info("Updating authority %q", dbAuth.Name)
			dbErr = db.Update(&dbAuth).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to insert authority %q: %w", dbAuth.Name, dbErr)
		}
	}

	return nil
}
