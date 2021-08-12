package backup

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportCertificates(logger *log.Logger, db database.ReadAccess, ownerType string,
	ownerID uint64) ([]file.Certificate, error) {

	var dbCerts model.Cryptos
	if err := db.Select(&dbCerts).Where("owner_type=? AND owner_id=?", ownerType,
		ownerID).Run(); err != nil {
		return nil, err
	}
	res := make([]file.Certificate, len(dbCerts))

	for i, src := range dbCerts {
		logger.Infof("Export Certificate %s\n", src.Name)
		cert := file.Certificate{
			Name:        src.Name,
			PublicKey:   src.SSHPublicKey,
			PrivateKey:  string(src.PrivateKey),
			Certificate: src.Certificate,
		}
		res[i] = cert
	}
	return res, nil
}
