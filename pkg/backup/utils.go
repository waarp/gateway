package backup

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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
		hash, _, err := auth.BcryptAuthHandler{}.ToDB(nil, pswd, "")

		return hash, err
	}

	hash, _, err := serializer.ToDB(nil, pswd, "")

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

//nolint:wrapcheck // wrapping errors adds nothing here
func restartService[T services.Service](list services.ServiceMap[T], obj database.Identifier,
	disabled bool, mkService func() (T, error),
) error {
	// Service already exists: restart if running, otherwise leave as is.
	state, exists := list.State(obj)
	if state == utils.StateRunning {
		return list.Restart(context.Background(), obj)
	} else if exists {
		return nil
	}

	// Service does not exist: add to list and start if not disabled.
	service, servErr := mkService()
	if servErr != nil {
		return servErr
	}

	list.Add(obj, service)
	if !disabled {
		_, err := list.Start(obj)

		return err
	}

	return nil
}
