package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importCerts(logger *log.Logger, db database.Access, list []file.Certificate,
	ownerType string, ownerID uint64) database.Error {

	for _, src := range list {
		// Create model with basic info to check existence
		var cert model.Cert

		//Check if cert exists
		exist := true
		err := db.Get(&cert, "owner_type=? AND owner_id=? AND name=?", ownerType,
			ownerID, src.Name).Run()
		if database.IsNotFound(err) {
			exist = false
		} else if err != nil {
			return err
		}

		// Populate
		cert.OwnerType = ownerType
		cert.OwnerID = ownerID
		cert.Name = src.Name
		cert.PrivateKey = []byte(src.PrivateKey)
		cert.PublicKey = []byte(src.PublicKey)
		cert.Certificate = []byte(src.Certificate)

		// Create/Update
		if exist {
			logger.Infof("Update certificate %s\n", cert.Name)
			err = db.Update(&cert).Run()
		} else {
			logger.Infof("Create certificate %s\n", cert.Name)
			err = db.Insert(&cert).Run()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
