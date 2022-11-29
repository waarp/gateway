package backup

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func importCerts(logger *log.Logger, db database.Access, list []file.Certificate,
	owner model.CryptoOwner,
) database.Error {
	for _, src := range list {
		// Create model with basic info to check existence
		var crypto model.Crypto

		// Check if crypto exists
		exist := true

		err := db.Get(&crypto, "name=?", src.Name).And(owner.GenCryptoSelectCond()).Run()
		if database.IsNotFound(err) {
			exist = false
		} else if err != nil {
			return err
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
			return err
		}
	}

	return nil
}
