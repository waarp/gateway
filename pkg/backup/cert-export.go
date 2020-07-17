package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportCertificates(logger *log.Logger, db *database.Session, ownerType string,
	ownerID uint64) ([]certificate, error) {

	dbCerts := []model.Cert{}
	filters := &database.Filters{
		Conditions: builder.And(
			builder.Eq{"owner_type": ownerType},
			builder.Eq{"owner_id": ownerID},
		),
	}
	if err := db.Select(&dbCerts, filters); err != nil {
		return nil, err
	}
	res := make([]certificate, len(dbCerts))

	for i, src := range dbCerts {
		logger.Infof("Export certificate %s\n", src.Name)
		cert := certificate{
			Name:        src.Name,
			PublicKey:   string(src.PublicKey),
			PrivateKey:  string(src.PrivateKey),
			Certificate: string(src.Certificate),
		}
		res[i] = cert
	}
	return res, nil
}
