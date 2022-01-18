package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

func exportCredentials(logger *log.Logger, db database.ReadAccess,
	owner model.CredOwnerTable,
) ([]file.Credential, []file.Certificate, string, error) {
	credentials, err := owner.GetCredentials(db)
	if err != nil {
		logger.Error("Failed to retrieve the %s's credentials: %s",
			owner.Appellation(), err)

		return nil, nil, "", fmt.Errorf("failed to retrieve the %s's credentials: %w",
			owner.Appellation(), err)
	}

	var pswd string

	fAuths := make([]file.Credential, len(credentials))
	certs := make([]file.Certificate, 0, len(credentials))

	for i, src := range credentials {
		fAuths[i] = file.Credential{
			Name:   src.Name,
			Type:   src.Type,
			Value:  src.Value,
			Value2: src.Value2,
		}

		exportLegacyCredentials(src, &certs, &pswd)
	}

	return fAuths, certs, pswd, nil
}

func exportLegacyCredentials(src *model.Credential, certs *[]file.Certificate,
	pswd *string,
) {
	switch src.Type {
	case auth.Password, auth.PasswordHash:
		*pswd = src.Value
	case auth.TLSTrustedCertificate:
		*certs = append(*certs, file.Certificate{
			Name:        src.Name,
			Certificate: src.Value,
		})
	case auth.TLSCertificate:
		*certs = append(*certs, file.Certificate{
			Name:        src.Name,
			Certificate: src.Value,
			PrivateKey:  src.Value2,
		})
	case sftp.AuthSSHPrivateKey:
		*certs = append(*certs, file.Certificate{
			Name:       src.Name,
			PrivateKey: src.Value,
		})
	case sftp.AuthSSHPublicKey:
		*certs = append(*certs, file.Certificate{
			Name:      src.Name,
			PublicKey: src.Value,
		})
	}
}
