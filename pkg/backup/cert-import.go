package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// Deprecated: to be replaced by importAuth.
func importCerts(logger *log.Logger, db database.Access, list []file.Certificate,
	owner model.CredOwnerTable,
) error {
	for _, src := range list {
		// Create model with basic info to check existence
		var crypto model.Credential

		// Check if crypto exists
		exist := true

		dbErr := db.Get(&crypto, "name=?", src.Name).And(owner.GetCredCond()).Run()
		if database.IsNotFound(dbErr) {
			exist = false
		} else if dbErr != nil {
			return fmt.Errorf("failed to retrieve certificate %q: %w", src.Name, dbErr)
		}

		// Populate
		owner.SetCredOwner(&crypto)
		crypto.Name = src.Name

		switch {
		case src.Certificate != "" && src.PrivateKey != "":
			if compatibility.IsLegacyR66CertPEM(src.Certificate) {
				crypto.Type = r66LegacyCert
			} else {
				crypto.Type = auth.TLSCertificate
				crypto.Value = src.Certificate
				crypto.Value2 = src.PrivateKey
			}
		case src.Certificate != "":
			if compatibility.IsLegacyR66CertPEM(src.Certificate) {
				crypto.Type = r66LegacyCert
			} else {
				crypto.Type = auth.TLSTrustedCertificate
				crypto.Value = src.Certificate
			}
		case src.PublicKey != "":
			crypto.Type = "ssh_public_key"
			crypto.Value = src.PublicKey
		case src.PrivateKey != "":
			crypto.Type = "ssh_private_key"
			crypto.Value = src.PrivateKey
		}

		// Create/Update
		if exist {
			logger.Infof("Update certificate %q", crypto.Name)
			dbErr = db.Update(&crypto).Run()
		} else {
			logger.Infof("Create certificate %q", crypto.Name)
			dbErr = db.Insert(&crypto).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import certificate %q: %w", crypto.Name, dbErr)
		}
	}

	return nil
}
