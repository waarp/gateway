package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportUsers(logger *log.Logger, db database.ReadAccess) ([]file.User, error) {
	var dbUsers model.Users
	if err := db.Select(&dbUsers).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve users: %w", err)
	}

	res := make([]file.User, len(dbUsers))

	for i, dbUser := range dbUsers {
		dbPerms := model.MaskToPerms(dbUser.Permissions)
		res[i] = file.User{
			Username:     dbUser.Username,
			PasswordHash: dbUser.PasswordHash,
			Permissions: file.Permissions{
				Transfers:      dbPerms.Transfers,
				Servers:        dbPerms.Servers,
				Partners:       dbPerms.Partners,
				Rules:          dbPerms.Rules,
				Users:          dbPerms.Users,
				Administration: dbPerms.Administration,
			},
		}

		logger.Info("Exported user %s\n", dbUser.Username)
	}

	return res, nil
}
