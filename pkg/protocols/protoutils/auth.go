package protoutils

import (
	"crypto/tls"
	"crypto/x509"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func CertificateAuthentication(db database.ReadAccess, logger *log.Logger,
	acc *model.LocalAccount, state *tls.ConnectionState,
) (bool, error) {
	if state == nil {
		logger.Debugf("No client certificate provided")

		return false, nil
	}

	return DoCertificateAuthentication(db, logger, acc, state.PeerCertificates)
}

func DoCertificateAuthentication(db database.ReadAccess, logger *log.Logger,
	acc *model.LocalAccount, chain []*x509.Certificate,
) (bool, error) {
	if len(chain) == 0 {
		logger.Debugf("No client certificate provided")

		return false, nil
	}

	if res, err := acc.Authenticate(db, auth.TLSTrustedCertificate, chain); err != nil {
		logger.Errorf("Error during remote certificate authentication for account %q: %v",
			acc.Login, err)

		return false, pipeline.NewErrorWith(err, types.TeInternal, "database error")
	} else if !res.Success {
		logger.Warningf("Failed to authenticate remote certificate for account %q: %v",
			acc.Login, res.Reason)

		return false, nil
	}

	return true, nil
}

func PasswordAuthentication(db database.ReadAccess, logger *log.Logger,
	acc *model.LocalAccount, password string,
) (bool, error) {
	if password == "" {
		logger.Debugf("No client certificate provided")

		return false, nil
	}

	if res, err := acc.Authenticate(db, auth.Password, password); err != nil {
		logger.Errorf("Error during password authentication for account %q: %v",
			acc.Login, err)

		return false, pipeline.NewErrorWith(err, types.TeInternal, "database error")
	} else if !res.Success {
		logger.Warningf("Failed to authenticate password for account %q: %v",
			acc.Login, res.Reason)

		return false, nil
	}

	return true, nil
}
