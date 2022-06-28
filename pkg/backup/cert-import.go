package backup

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func importCerts(logger *log.Logger, db database.Access, list []file.Certificate,
	ownerType string, ownerID uint64,
) database.Error {
	for _, src := range list {
		// Create model with basic info to check existence
		var crypto model.Crypto

		// Check if crypto exists
		exist := true
		err := db.Get(&crypto, "owner_type=? AND owner_id=? AND name=?", ownerType,
			ownerID, src.Name).Run()

		if database.IsNotFound(err) {
			exist = false
		} else if err != nil {
			return err
		}

		// Populate
		crypto.OwnerType = ownerType
		crypto.OwnerID = ownerID
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
