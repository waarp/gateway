package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func PreprocessImport(data *file.Data) error {
	if err := preprocessServers(data.Locals); err != nil {
		return err
	}

	if err := preprocessPartners(data.Remotes); err != nil {
		return err
	}

	return preprocessUsers(data.Users)
}

func preprocessUsers(users []file.User) error {
	var err error
	for i := range users {
		user := &users[i]

		if user.PasswordHash == "" && user.Password != "" {
			if user.PasswordHash, err = utils.HashPassword(database.BcryptRounds,
				user.Password); err != nil {
				return fmt.Errorf("failed to hash password for user %q: %w", user.Username, err)
			}
		}
	}

	return nil
}

func preprocessServers(servers []file.LocalAgent) error {
	var err error
	for _, server := range servers {
		if err = preprocessLocalAccounts(server.Accounts); err != nil {
			return err
		}
	}

	return nil
}

func preprocessLocalAccounts(accounts []file.LocalAccount) error {
	for i := range accounts {
		account := &accounts[i]

		hasPswd, err := preprocessPasswordHashes(account.Credentials)
		if err != nil {
			return err
		}

		if hasPswd {
			continue
		}

		if account.PasswordHash != "" {
			err = addPswdHashCred(&account.Credentials, account.PasswordHash)
		} else if account.Password != "" {
			err = addPswdHashCred(&account.Credentials, account.Password)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func preprocessPartners(partners []file.RemoteAgent) error {
	for i := range partners {
		partner := &partners[i]

		hasPswd, err := preprocessPasswordHashes(partner.Credentials)
		if err != nil {
			return err
		}

		if isR66(partner.Protocol) && !hasPswd {
			var confPswd string
			if confPswd, err = utils.GetAs[string](partner.Configuration, "serverPassword"); err == nil {
				if err = addPswdHashCred(&partner.Credentials, confPswd); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
