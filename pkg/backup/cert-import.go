package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

func importCerts(logger *log.Logger, db database.Access, list []file.Certificate,
	ownerType string, ownerID uint64) database.Error {

	for _, src := range list {
		// Create model with basic info to check existence
		var crypto model.Crypto

		//Check if crypto exists
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
			logger.Infof("Update certificate %s\n", crypto.Name)
			err = db.Update(&crypto).Run()
		} else {
			logger.Infof("Create certificate %s\n", crypto.Name)
			err = db.Insert(&crypto).Run()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
