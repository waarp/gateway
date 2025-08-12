package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errNoValidCert = errors.New("could not find a valid certificate for HTTP server")

func (h *httpService) makeTLSConf(*tls.ClientHelloInfo) (*tls.Config, error) {
	creds, dbErr := h.agent.GetCredentials(h.db, auth.TLSCertificate)
	if dbErr != nil {
		h.logger.Errorf("Failed to retrieve server certificates: %s", dbErr)

		return nil, fmt.Errorf("failed to retrieve server certificates: %w", dbErr)
	}

	var tlsCerts []tls.Certificate

	for _, cred := range creds {
		cert, err := tls.X509KeyPair([]byte(cred.Value), []byte(cred.Value2))
		if err != nil {
			h.logger.Warningf("Failed to parse server certificate: %v", err)

			continue
		}

		tlsCerts = append(tlsCerts, cert)
	}

	if len(tlsCerts) == 0 {
		h.logger.Error("Could not find a valid certificate for HTTP server")

		return nil, errNoValidCert
	}

	return &tls.Config{
		MinVersion:            h.conf.MinTLSVersion.TLS(),
		Certificates:          tlsCerts,
		ClientAuth:            tls.RequestClientCert,
		VerifyPeerCertificate: auth.VerifyClientCert(h.db, h.logger, h.agent),
		VerifyConnection:      compatibility.LogSha1(h.logger),
	}, nil
}

func (h *httpService) listen() error {
	addr := conf.GetRealAddress(h.agent.Host(),
		utils.FormatUint(h.agent.Address.Port))

	var (
		list   net.Listener
		netErr error
	)

	if h.agent.Protocol == HTTPS {
		list, netErr = tls.Listen("tcp", addr, &tls.Config{
			MinVersion:         h.conf.MinTLSVersion.TLS(),
			GetConfigForClient: h.makeTLSConf,
		})
	} else {
		list, netErr = net.Listen("tcp", addr)
	}

	if netErr != nil {
		h.logger.Errorf("Failed to start server listener: %s", netErr)

		return fmt.Errorf("failed to start server listener: %w", netErr)
	}

	go func() {
		servErr := h.serv.Serve(list)
		if !errors.Is(servErr, http.ErrServerClosed) {
			h.logger.Errorf("Unexpected error: %v", servErr)
			h.state.Set(utils.StateError, fmt.Sprintf("unexpected error: %v", servErr))
		} else {
			h.state.Set(utils.StateOffline, "")
		}
	}()

	return nil
}

//nolint:contextcheck //would be too complicated to change
func (h *httpService) makeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.checkShutdown(w) {
			return
		}

		acc, canContinue := h.checkAuthent(w, r)
		if !canContinue {
			return
		}

		handler := &httpHandler{
			agent:   h.agent,
			account: acc,
			tracer:  h.tracer,
			db:      h.db,
			logger:  h.logger,
			req:     r,
			resp:    w,
		}

		//nolint:contextcheck //context is already passed in the request itself
		switch r.Method {
		case http.MethodPost:
			handler.handle(false)
		case http.MethodGet:
			handler.handle(true)
		case http.MethodHead:
			handler.handleHead()
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

//nolint:funlen //function is fine for now
func (h *httpService) checkAuthent(w http.ResponseWriter, r *http.Request,
) (*model.LocalAccount, bool) {
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
	if err := h.db.Get(&acc, "login=? AND local_agent_id=?", login, h.agent.ID).
		Run(); err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("Failed to retrieve user credentials: %v", err)
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
			h.logger.Warningf("Mismatched login %q and certificate subject %q", login, cn)
			unauthorized(w, "auth: mismatched login and certificate subject")

			return nil, false
		}

		authentified = true
	}

	if pswd != "" {
		if res, err := acc.Authenticate(h.db, h.agent, auth.Password, pswd); err != nil {
			h.logger.Errorf("Failed to check password for user %q: %v", acc.Login, err)
			http.Error(w, "internal authentication error", http.StatusInternalServerError)

			return nil, false
		} else if !res.Success {
			h.logger.Warningf("Invalid credentials for user %q: %s", acc.Login, res.Reason)
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

func (h *httpService) checkShutdown(w http.ResponseWriter) bool {
	select {
	case <-h.shutdown:
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("server is shutting down")) //nolint:errcheck // error is irrelevant at this point

		return true
	default:
		return false
	}
}
