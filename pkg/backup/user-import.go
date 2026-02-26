package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:funlen //splitting would add complexity
func importUsers(logger *log.Logger, db database.Access, users []file.User,
	reset bool,
) error {
	for i := range users {
		if reset {
			var dbUsers model.Users
			if err := db.Select(&dbUsers).Run(); err != nil {
				return fmt.Errorf("failed to retrieve existing users: %w", err)
			}

			for _, dbUser := range dbUsers {
				if err := db.Delete(dbUser).Run(); err != nil {
					return fmt.Errorf("failed to delete user %q: %w", dbUser.Username, err)
				}
			}
		}

		var (
			user   = &users[i]
			dbUser model.User
		)

		exists := true

		if err := db.Get(&dbUser, "username=? AND owner=?", user.Username,
			conf.GlobalConfig.GatewayName).Run(); err != nil {
			if !database.IsNotFound(err) {
				return fmt.Errorf("failed to retrieve user %q: %w", dbUser.Username, err)
			}

			exists = false
		}

		dbUser.Username = user.Username

		if user.PasswordHash != "" {
			dbUser.PasswordHash = user.PasswordHash
		}

		var err error
		if dbUser.Permissions, err = model.PermsToMask(&model.Permissions{
			Transfers:      user.Permissions.Transfers,
			Servers:        user.Permissions.Servers,
			Partners:       user.Permissions.Partners,
			Rules:          user.Permissions.Rules,
			Users:          user.Permissions.Users,
			Administration: user.Permissions.Administration,
		}); err != nil {
			return fmt.Errorf("failed to parse user %q's permissions: %w", dbUser.Username, err)
		}

		var msg string

		if exists {
			msg = fmt.Sprintf("Updated user %s\n", dbUser.Username)
			err = db.Update(&dbUser).Run()
		} else {
			msg = fmt.Sprintf("Created user %s\n", dbUser.Username)
			err = db.Insert(&dbUser).Run()
		}

		if err != nil {
			return fmt.Errorf("failed to import user %q: %w", dbUser.Username, err)
		}

		logger.Info(msg)
	}

	return nil
}
