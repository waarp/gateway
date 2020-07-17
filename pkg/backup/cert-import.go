package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importCerts(logger *log.Logger, db *database.Session, list []certificate,
	ownerType string, ownerID uint64) error {

	for _, src := range list {
		// Create model with basic info to check existence
		cert := &model.Cert{
			OwnerType: ownerType,
			OwnerID:   ownerID,
			Name:      src.Name,
		}

		//Check if cert exists
		exist := true
		err := db.Get(cert)
		if err != nil {
			if err == database.ErrNotFound {
				exist = false
			} else {
				return err
			}
		}

		// Populate
		cert.PrivateKey = []byte(src.PrivateKey)
		cert.PublicKey = []byte(src.PublicKey)
		cert.Certificate = []byte(src.Certificate)

		// Create/Update
		if exist {
			logger.Infof("Update certificate %s\n", cert.Name)
			err = db.Update(cert, cert.ID, false)
		} else {
			logger.Infof("Create certificate %s\n", cert.Name)
			err = db.Create(cert)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
