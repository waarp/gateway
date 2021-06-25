package r66

import (
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-r66/r66"
)

type authHandler struct {
	*Service
}

func (a *authHandler) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) {
	select {
	case <-a.shutdown:
		return nil, sigShutdown
	default:
	}

	if auth.FinalHash && !strings.EqualFold(auth.Digest, "SHA-256") {
		a.logger.Warningf("Unknown hash digest '%s'", auth.Digest)
		return nil, internal.NewR66Error(r66.Unimplemented, "unknown final hash digest")
	}

	var certAcc, pwdAcc *model.LocalAccount
	var err *r66.Error
	if certAcc, err = a.certAuth(auth); err != nil {
		return nil, err
	}
	if pwdAcc, err = a.passwordAuth(auth); err != nil {
		return nil, err
	}
	if certAcc == nil && pwdAcc == nil {
		return nil, internal.NewR66Error(r66.BadAuthent, "missing credentials")
	}

	acc := certAcc
	if certAcc == nil {
		acc = pwdAcc
	} else if pwdAcc != nil && certAcc.ID != pwdAcc.ID {
		return nil, internal.NewR66Error(r66.BadAuthent, "the given certificate does not match the given login")
	}

	ses := sessionHandler{
		authHandler: a,
		account:     acc,
		conf:        auth,
	}
	return &ses, nil
}

func (a *authHandler) certAuth(auth *r66.Authent) (*model.LocalAccount, *r66.Error) {
	if auth.TLS == nil || len(auth.TLS.PeerCertificates) == 0 {
		return nil, nil
	}

	sign := utils.MakeSignature(auth.TLS.PeerCertificates[0])
	var crypto model.Crypto
	if err := a.db.Get(&crypto, "owner_type=? AND signature=?", "local_accounts",
		sign).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, internal.NewR66Error(r66.BadAuthent, "unknown certificate")
		}
		a.logger.Errorf("Failed to retrieve client certificate: %s", err)
		return nil, r66ErrDatabase
	}

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "id=?", crypto.OwnerID).Run(); err != nil {
		a.logger.Errorf("Failed to retrieve client account: %s", err)
		return nil, r66ErrDatabase
	}
	return &acc, nil
}

func (a *authHandler) passwordAuth(auth *r66.Authent) (*model.LocalAccount, *r66.Error) {
	if auth.Login == "" || len(auth.Password) == 0 {
		return nil, nil
	}

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "login=? AND local_agent_id=?", auth.Login,
		a.agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			a.logger.Errorf("Failed to retrieve credentials from database: %s", err)
			return nil, r66ErrDatabase
		}
	}

	if bcrypt.CompareHashAndPassword(acc.PasswordHash, auth.Password) != nil {
		if acc.Login == "" {
			a.logger.Warningf("Authentication failed with unknown account '%s'", auth.Login)
		} else {
			a.logger.Warningf("Account '%s' authenticated with wrong password %s",
				auth.Login, string(auth.Password))
		}
		return nil, internal.NewR66Error(r66.BadAuthent, "incorrect credentials")
	}
	return &acc, nil
}
