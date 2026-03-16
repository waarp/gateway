package backup

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

const (
	r66           = "r66"
	r66TLS        = "r66-tls"
	r66LegacyCert = "r66_legacy_certificate"
)

//nolint:wrapcheck //no need to wrap here
func hashPswd(pswd, protocol string) (string, error) {
	handler := authentication.GetInternalAuthHandler(auth.Password, protocol)
	serializer, ok := handler.(authentication.Serializer)
	if !ok {
		hash, _, err := auth.BcryptAuthHandler{}.ToDB(pswd, "")

		return hash, err
	}

	hash, _, err := serializer.ToDB(pswd, "")

	return hash, err
}

func pswdCred(value string) file.Credential {
	return file.Credential{
		Name:  auth.Password,
		Type:  auth.Password,
		Value: value,
	}
}

func addPswdHashCred(creds *[]file.Credential, pswd, protocol string) error {
	hash, err := hashPswd(pswd, protocol)
	if err != nil {
		return err
	}

	*creds = append(*creds, pswdCred(hash))

	return nil
}

func preprocessPasswordHashes(creds []file.Credential, protocol string) (bool, error) {
	var (
		err     error
		hasPswd bool
	)
	for i := range creds {
		cred := &creds[i]
		if cred.Type == auth.Password {
			hasPswd = true
			if cred.Value, err = hashPswd(cred.Value, protocol); err != nil {
				return hasPswd, err
			}
		}
	}

	return hasPswd, nil
}

func isR66(proto string) bool {
	return proto == r66 || proto == r66TLS
}
