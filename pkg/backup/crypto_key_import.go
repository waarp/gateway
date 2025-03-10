package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importCryptoKeys(logger *log.Logger, db database.Access, keys []*file.CryptoKey,
	reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.CryptoKey{}).Run(); err != nil {
			return fmt.Errorf("failed to purge cryptographic keys: %w", err)
		}
	}

	for _, key := range keys {
		var (
			dbKey model.CryptoKey
			isNew bool
		)

		if err := db.Get(&dbKey, "name=?", key.Name).Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf("failed to retrieve crypto key %q: %w", key.Name, err)
		}

		dbKey.Name = key.Name
		dbKey.Type = key.Type
		dbKey.Key = database.SecretText(key.Key)

		var dbErr error

		if isNew {
			logger.Info("Inserting new crypto key %q", dbKey.Name)
			dbErr = db.Insert(&dbKey).Run()
		} else {
			logger.Info("Updating existing crypto key %q", dbKey.Name)
			dbErr = db.Update(&dbKey).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import crypto key %q: %w",
				dbKey.Name, dbErr)
		}
	}

	return nil
}
