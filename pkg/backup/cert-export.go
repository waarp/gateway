package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func exportCertificates(logger *log.Logger, db database.ReadAccess, ownerType string,
	ownerID uint64) ([]file.Certificate, error) {

	var dbCerts model.Certificates
	if err := db.Select(&dbCerts).Where("owner_type=? AND owner_id=?", ownerType,
		ownerID).Run(); err != nil {
		return nil, err
	}
	res := make([]file.Certificate, len(dbCerts))

	for i, src := range dbCerts {
		logger.Infof("Export Certificate %s\n", src.Name)
		cert := file.Certificate{
			Name:        src.Name,
			PublicKey:   string(src.PublicKey),
			PrivateKey:  string(src.PrivateKey),
			Certificate: string(src.Certificate),
		}
		res[i] = cert
	}
	return res, nil
}
