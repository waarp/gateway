package backup

import (
	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportCertificates(logger *log.Logger, db *database.Session, ownerType string,
	ownerID uint64) ([]Certificate, error) {

	var dbCerts []model.Cert
	filters := &database.Filters{
		Conditions: builder.And(
			builder.Eq{"owner_type": ownerType},
			builder.Eq{"owner_id": ownerID},
		),
	}
	if err := db.Select(&dbCerts, filters); err != nil {
		return nil, err
	}
	res := make([]Certificate, len(dbCerts))

	for i, src := range dbCerts {
		logger.Infof("Export Certificate %s\n", src.Name)
		cert := Certificate{
			Name:        src.Name,
			PublicKey:   string(src.PublicKey),
			PrivateKey:  string(src.PrivateKey),
			Certificate: string(src.Certificate),
		}
		res[i] = cert
	}
	return res, nil
}
