package r66

import (
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/throttle"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

type authHandler struct {
	*service
}

func (a *authHandler) ValidAuth(authent *r66.Authent) (r66.SessionHandler, error) {
	if authent == nil || authent.Login == "" {
		return nil, internal.NewR66Error(r66.BadAuthent, "missing credentials")
	}

	a.logger.Debugf("Connection received from %s", authent.Login)

	attempt := throttle.Default().Attempt(&a.db.Config.Security, R66+":"+a.dbAgent.Name,
		authent.Login, authent.Address)

	if attempt.Locked() {
		a.logger.Warningf("Authentication of account %q from %q refused: too many failed attempts",
			authent.Login, authent.Address)

		return nil, internal.NewR66Error(r66.BadAuthent, "authentication failed")
	}

	acc, accErr := a.dbAgent.GetAccount(a.db, authent.Login)
	if accErr != nil && !database.IsNotFound(accErr) {
		a.logger.Errorf("Failed to retrieve client account: %v", accErr)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	var authenticated bool

	if certAuthenticated, err := a.certAuth(authent, acc); err != nil {
		return nil, err
	} else if certAuthenticated {
		authenticated = true
	}

	if pwsdAuthenticated, err := a.passwordAuth(authent, acc); err != nil {
		return nil, err
	} else if pwsdAuthenticated {
		authenticated = true
	}

	// A disallowed address is refused like invalid credentials, so that the
	// answer does not tell whether the credentials were valid. The actual
	// reason is logged by CheckIP.
	if !authenticated || !acc.CheckIP(a.logger, authent.Address) {
		a.logger.Warningf("Authentication failed for account %q", authent.Login)

		if attempt.Fail() {
			a.logger.Warningf("Too many failed authentications for account %q from %q, "+
				"new attempts are temporarily refused", authent.Login, authent.Address)
		}

		return nil, internal.NewR66Error(r66.BadAuthent, "authentication failed")
	}

	authent.Filesize = true

	if authent.FinalHash = authent.FinalHash && !a.r66Conf.NoFinalHash; authent.FinalHash {
		if _, err := internal.GetHasher(authent.Digest); err != nil {
			return nil, internal.NewR66Error(r66.BadAuthent, "unsupported hash algorithm")
		}

		if len(a.r66Conf.FinalHashAlgos) != 0 && !utils.ContainsOneOf(a.r66Conf.FinalHashAlgos, authent.Digest) {
			return nil, internal.NewR66Error(r66.BadAuthent, "unauthorized hash algorithm")
		}
	} else {
		authent.Digest = ""
	}

	attempt.Succeed()

	return &sessionHandler{
		authHandler: a,
		account:     acc,
		conf:        authent,
	}, nil
}

func (a *authHandler) certAuth(authent *r66.Authent, acc *model.LocalAccount,
) (bool, *r66.Error) {
	if authent.TLS == nil || len(authent.TLS.PeerCertificates) == 0 {
		return false, nil
	}

	// If account uses legacy certificate, check if the certificate is a legacy one.
	if r66auth.UsesLegacyCert(a.db, acc) {
		return compatibility.IsLegacyR66Cert(authent.TLS.PeerCertificates[0]), nil
	}

	// Otherwise, do normal certificate authentication.
	success, err := protoutils.CertificateAuthentication(a.db, a.logger, acc, authent.TLS)
	if err != nil {
		return false, internal.NewR66Error(r66.Internal, "internal authentication error")
	} else if !success {
		return false, nil
	}

	return true, nil
}

func (a *authHandler) passwordAuth(authent *r66.Authent, acc *model.LocalAccount,
) (bool, *r66.Error) {
	success, err := protoutils.PasswordAuthentication(a.db, a.logger, acc, string(authent.Password))
	if err != nil {
		return false, internal.NewR66Error(r66.Internal, "internal authentication error")
	} else if !success {
		return false, nil
	}

	return true, nil
}
