package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportCryptoKeys(logger *log.Logger, db database.ReadAccess) ([]*file.CryptoKey, error) {
	var dbKeys model.CryptoKeys
	if err := db.Select(&dbKeys).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve crypto keys: %w", err)
	}

	keys := make([]*file.CryptoKey, len(dbKeys))

	for i, dbKey := range dbKeys {
		keys[i] = &file.CryptoKey{
			Name: dbKey.Name,
			Type: dbKey.Type,
			Key:  dbKey.Key.String(),
		}

		logger.Infof("Exported crypto key %q", dbKey.Name)
	}

	return keys, nil
}
