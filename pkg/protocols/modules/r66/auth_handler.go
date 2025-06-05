package r66

import (
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
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

	acc := model.LocalAccount{Login: authent.Login}
	if err := a.db.Get(&acc, "local_agent_id=? AND login=?", a.agent.ID,
		authent.Login).Run(); err != nil && !database.IsNotFound(err) {
		a.logger.Errorf("Failed to retrieve client account: %v", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	if len(acc.IPAddresses) > 0 {
		remoteIP := protoutils.GetIP(authent.Address)
		if !acc.IPAddresses.Contains(remoteIP) {
			return nil, internal.NewR66Error(r66.BadAuthent, "unauthorized IP address")
		}
	}

	var authenticated bool

	if certAuthenticated, err := a.certAuth(authent, &acc); err != nil {
		return nil, err
	} else if certAuthenticated {
		authenticated = true
	}

	if pwsdAuthenticated, err := a.passwordAuth(authent, &acc); err != nil {
		return nil, err
	} else if pwsdAuthenticated {
		authenticated = true
	}

	if !authenticated {
		a.logger.Warningf("Authentication failed for account %q", authent.Login)

		return nil, internal.NewR66Error(r66.BadAuthent, "authentication failed")
	}

	authent.Filesize = true

	if authent.FinalHash = (authent.FinalHash && !a.r66Conf.NoFinalHash); authent.FinalHash {
		if _, err := internal.GetHasher(authent.Digest); err != nil {
			return nil, internal.NewR66Error(r66.BadAuthent, "unsuported hash algorithm")
		}

		if len(a.r66Conf.FinalHashAlgos) != 0 && !utils.ContainsOneOf(a.r66Conf.FinalHashAlgos, authent.Digest) {
			return nil, internal.NewR66Error(r66.BadAuthent, "unauthorized hash algorithm")
		}
	} else {
		authent.Digest = ""
	}

	return &sessionHandler{
		authHandler: a,
		account:     &acc,
		conf:        authent,
	}, nil
}

func (a *authHandler) certAuth(authent *r66.Authent, acc *model.LocalAccount,
) (bool, *r66.Error) {
	if authent.TLS == nil || len(authent.TLS.PeerCertificates) == 0 {
		return false, nil
	}

	// If client send a legacy certificate, check if it's allowed (both globally
	// and for this account).
	if compatibility.IsLegacyR66Cert(authent.TLS.PeerCertificates[0]) {
		if n, err := a.db.Count(&model.Credential{}).Where(acc.GetCredCond()).
			Where("type=?", AuthLegacyCertificate).Run(); err != nil {
			return false, internal.NewR66Error(r66.Internal, "database error")
		} else if n == 0 {
			return false, internal.NewR66Error(r66.BadAuthent, "invalid certificate")
		}

		return true, nil
	}

	// If client send a "normal" certificate, check if the Common Name matches
	// the R66 login.
	if cn := authent.TLS.PeerCertificates[0].Subject.CommonName; cn != acc.Login {
		return false, internal.NewR66Error(r66.BadAuthent,
			"the certificate's Common Name does not match the R66 login")
	}

	return true, nil
}

func (a *authHandler) passwordAuth(authent *r66.Authent, acc *model.LocalAccount,
) (bool, *r66.Error) {
	if len(authent.Password) == 0 {
		return false, nil
	}

	if res, err := acc.Authenticate(a.db, a.agent, auth.Password, authent.Password); err != nil {
		a.logger.Errorf("Failed to authenticate account %q: %v", acc.Login, err)

		return false, internal.NewR66Error(r66.Internal, "internal authentication error")
	} else if !res.Success {
		a.logger.Warningf("Authentication failed for account %q: %s", authent.Login, res.Reason)

		return false, internal.NewR66Error(r66.BadAuthent, "authentication failed")
	}

	return true, nil
}
