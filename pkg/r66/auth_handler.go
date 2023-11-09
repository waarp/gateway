package r66

import (
	"strings"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
)

type authHandler struct {
	*Service
}

func (a *authHandler) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) {
	select {
	case <-a.shutdown:
		return nil, internal.NewR66Error(r66.Shutdown, "service is shutting down")
	default:
	}

	if auth.FinalHash && !strings.EqualFold(auth.Digest, "SHA-256") {
		a.logger.Warning("Unknown hash digest '%s'", auth.Digest)

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

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "local_agent_id=? AND login=?", a.agentID, auth.Login).Run(); err != nil {
		if !database.IsNotFound(err) {
			a.logger.Error("Failed to retrieve client account: %s", err)

			return nil, internal.NewR66Error(r66.Internal, "database error")
		}
	}

	var cryptos model.Cryptos
	if err := a.db.Select(&cryptos).Where("local_account_id=?", acc.ID).Run(); err != nil {
		a.logger.Error("Failed to retrieve client certificates: %s", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	if err := model.CheckClientAuthent(&cryptos, acc.Login, auth.TLS.PeerCertificates); err != nil {
		a.logger.Warning(err.Error())

		return nil, &r66.Error{Code: r66.BadAuthent, Detail: err.Error()}
	}

	return &acc, nil
}

func (a *authHandler) passwordAuth(auth *r66.Authent) (*model.LocalAccount, *r66.Error) {
	if auth.Login == "" || len(auth.Password) == 0 {
		return nil, nil
	}

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "login=? AND local_agent_id=?", auth.Login,
		a.agentID).Run(); err != nil {
		if !database.IsNotFound(err) {
			a.logger.Error("Failed to retrieve credentials from database: %s", err)

			return nil, internal.NewR66Error(r66.Internal, "database error")
		}
	}

	if bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), auth.Password) != nil {
		if acc.Login == "" {
			a.logger.Warning("Authentication failed with unknown account '%s'", auth.Login)
		} else {
			a.logger.Warning("Account '%s' authenticated with wrong password", auth.Login)
		}

		return nil, internal.NewR66Error(r66.BadAuthent, "incorrect credentials")
	}

	return &acc, nil
}
