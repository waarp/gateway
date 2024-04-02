package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func importCerts(logger *log.Logger, db database.Access, list []file.Certificate,
	owner model.CryptoOwner,
) error {
	for _, src := range list {
		// Create model with basic info to check existence
		var crypto model.Crypto

		// Check if crypto exists
		exist := true

		err := db.Get(&crypto, "name=?", src.Name).And(owner.GenCryptoSelectCond()).Run()
		if database.IsNotFound(err) {
			exist = false
		} else if err != nil {
			return fmt.Errorf("failed to retrieve certificate %q: %w", src.Name, err)
		}

		// Populate
		owner.SetCryptoOwner(&crypto)
		crypto.Name = src.Name
		crypto.PrivateKey = types.CypherText(src.PrivateKey)
		crypto.SSHPublicKey = src.PublicKey
		crypto.Certificate = src.Certificate

		// Create/Update
		if exist {
			logger.Info("Update certificate %s\n", crypto.Name)
			err = db.Update(&crypto).Run()
		} else {
			logger.Info("Create certificate %s\n", crypto.Name)
			err = db.Insert(&crypto).Run()
		}

		if err != nil {
			return fmt.Errorf("failed to import certificate %q: %w", crypto.Name, err)
		}
	}

	return nil
}
