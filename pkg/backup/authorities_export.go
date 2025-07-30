package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportAuthorities(logger *log.Logger, db database.ReadAccess) ([]*file.Authority, error) {
	var authorities model.Authorities
	if err := db.Select(&authorities).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve authorities: %w", err)
	}

	var res []*file.Authority
	if len(authorities) > 0 {
		res = make([]*file.Authority, len(authorities))
	}

	for i, authority := range authorities {
		logger.Infof("Exported authority %q", authority.Name)
		res[i] = &file.Authority{
			Name:           authority.Name,
			Type:           authority.Type,
			PublicIdentity: authority.PublicIdentity,
			ValidHosts:     authority.ValidHosts,
		}
	}

	return res, nil
}
