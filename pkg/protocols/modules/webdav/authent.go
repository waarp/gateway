package webdav

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

func (s *server) auth(w http.ResponseWriter, r *http.Request) (*model.LocalAccount, bool) {
	// OPTIONS request don't need authentication
	if r.Method == http.MethodOptions {
		return &model.LocalAccount{}, true
	}

	var (
		acc          model.LocalAccount
		authentified bool
	)

	login, pswd, ok := r.BasicAuth()
	if !ok || login == "" {
		unauthorized(w, "auth: missing login")

		return nil, false
	}

	acc.Login = login

	// We purposefully ignore NotFound errors to avoid leaking information
	// about the existence of an account.
	if err := s.db.Get(&acc, "login=? AND local_agent_id=?", login, s.agent.ID).
		Run(); err != nil && !database.IsNotFound(err) {
		s.logger.Errorf("Failed to retrieve user credentials: %v", err)
		http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

		return nil, false
	}

	if len(acc.IPAddresses) > 0 {
		remoteIP := protoutils.GetIP(r.RemoteAddr)
		if !acc.IPAddresses.Contains(remoteIP) {
			http.Error(w, "Unauthorized IP address", http.StatusUnauthorized)

			return nil, false
		}
	}

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		if cn := r.TLS.PeerCertificates[0].Subject.CommonName; cn != login {
			s.logger.Warningf("Mismatched login %q and certificate subject %q", login, cn)
			unauthorized(w, "auth: mismatched login and certificate subject")

			return nil, false
		}

		authentified = true
	}

	if pswd != "" {
		if res, err := acc.Authenticate(s.db, s.agent, auth.Password, pswd); err != nil {
			s.logger.Errorf("Failed to check password for user %q: %v", acc.Login, err)
			http.Error(w, "internal authentication error", http.StatusInternalServerError)

			return nil, false
		} else if !res.Success {
			s.logger.Warningf("Invalid credentials for user %q: %s", acc.Login, res.Reason)
			unauthorized(w, "auth: invalid credentials")

			return nil, false
		}

		authentified = true
	}

	if !authentified {
		unauthorized(w, "missing credentials")

		return nil, false
	}

	return &acc, true
}
