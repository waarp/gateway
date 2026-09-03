package webdav

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

func (s *server) auth(w http.ResponseWriter, r *http.Request) (*model.LocalAccount, bool) {
	// OPTIONS request don't need authentication
	if r.Method == http.MethodOptions {
		return &model.LocalAccount{}, true
	}

	var authenticated bool
	login, pswd, ok := r.BasicAuth()
	if !ok || login == "" {
		unauthorized(w, "auth: missing login")

		return nil, false
	}

	// We purposefully ignore NotFound errors to avoid leaking information
	// about the existence of an account.
	acc, accErr := s.agent.GetAccount(s.db, login)
	if accErr != nil && !database.IsNotFound(accErr) {
		s.logger.Errorf("Failed to retrieve user credentials: %v", accErr)
		http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

		return nil, false
	}

	if success, err := protoutils.CertificateAuthentication(s.db, s.logger, acc, r.TLS); err != nil {
		s.logger.Errorf("Failed to check certificate for user %q: %v", acc.Login, err)
		http.Error(w, "internal authentication error", http.StatusInternalServerError)

		return nil, false
	} else if !success {
		authenticated = true
	}

	if success, err := protoutils.PasswordAuthentication(s.db, s.logger, acc, pswd); err != nil {
		s.logger.Errorf("Failed to check password for user %q: %v", acc.Login, err)
		http.Error(w, "internal authentication error", http.StatusInternalServerError)

		return nil, false
	} else if success {
		authenticated = true
	}

	if !authenticated {
		unauthorized(w, "missing credentials")

		return nil, false
	}

	if !acc.CheckIP(s.logger, r.RemoteAddr) {
		unauthorized(w, "invalid IP address")

		return nil, false
	}

	return acc, true
}
