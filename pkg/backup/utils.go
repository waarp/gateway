package backup

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func hashPswd(pswd string) (string, error) {
	return utils.HashPassword(database.BcryptRounds, pswd)
}

func pswdCred(value string) file.Credential {
	return file.Credential{
		Name:  auth.Password,
		Type:  auth.Password,
		Value: value,
	}
}

func addPswdHashCred(creds *[]file.Credential, pswd string) error {
	hash, err := hashPswd(pswd)
	if err != nil {
		return err
	}

	*creds = append(*creds, pswdCred(hash))

	return nil
}

func preprocessPasswordHashes(creds []file.Credential) (bool, error) {
	var (
		err     error
		hasPswd bool
	)
	for i := range creds {
		cred := &creds[i]
		if cred.Type == auth.Password {
			hasPswd = true
			if cred.Value, err = hashPswd(cred.Value); err != nil {
				return hasPswd, err
			}
		}
	}

	return hasPswd, nil
}

func isR66(proto string) bool {
	return proto == "r66" || proto == "r66-tls"
}
