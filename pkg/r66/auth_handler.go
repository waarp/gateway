package r66

import (
	"strings"

	"code.waarp.fr/waarp-r66/r66"
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

	var acc model.LocalAccount
	if err := a.db.Get(&acc, "local_agent_id=? AND login=?", a.agent.ID, auth.Login).Run(); err != nil {
		if !database.IsNotFound(err) {
			a.logger.Errorf("Failed to retrieve client account: %s", err)

			return nil, internal.NewR66Error(r66.Internal, "database error")
		}
	}

	var cryptos model.Cryptos
	if err := a.db.Select(&cryptos).Where("owner_type=? AND owner_id=?",
		acc.TableName(), acc.ID).Run(); err != nil {
		a.logger.Errorf("Failed to retrieve client certificates: %s", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	if err := cryptos.CheckClientAuthent(auth.Login, auth.TLS.PeerCertificates); err != nil {
		a.logger.Warningf(err.Error())

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
		a.agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			a.logger.Errorf("Failed to retrieve credentials from database: %s", err)

			return nil, internal.NewR66Error(r66.Internal, "database error")
		}
	}

	if bcrypt.CompareHashAndPassword(acc.PasswordHash, auth.Password) != nil {
		if acc.Login == "" {
			a.logger.Warningf("Authentication failed with unknown account '%s'", auth.Login)
		} else {
			a.logger.Warningf("Account '%s' authenticated with wrong password", auth.Login)
		}

		return nil, internal.NewR66Error(r66.BadAuthent, "incorrect credentials")
	}

	return &acc, nil
}
