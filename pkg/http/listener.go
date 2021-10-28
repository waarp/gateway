package http

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errNoValidCert = errors.New("could not find a valid certificate for HTTP server")

func (h *httpService) makeTLSConf(agent *model.LocalAgent) (*tls.Config, error) {
	var certs model.Cryptos
	if err := h.db.Select(&certs).Where("owner_type=? AND owner_id=?", agent.TableName(),
		agent.ID).Run(); err != nil {
		h.logger.Error("Failed to retrieve server certificates: %s", err)

		return nil, fmt.Errorf("failed to retrieve server certificates: %w", err)
	}

	var tlsCerts []tls.Certificate

	for _, c := range certs {
		cert, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey))
		if err != nil {
			h.logger.Warning("Failed to parse server certificate: %s", err)

			continue
		}

		tlsCerts = append(tlsCerts, cert)
	}

	if len(tlsCerts) == 0 {
		h.logger.Error("Could not find a valid certificate for HTTP server")

		return nil, errNoValidCert
	}

	var clientCerts model.Cryptos
	if err := h.db.Select(&clientCerts).Where("owner_type='local_accounts' AND "+
		"owner_id IN (SELECT id FROM local_accounts WHERE local_agent_id=?)",
		agent.ID).Run(); err != nil {
		h.logger.Error("Failed to retrieve client certificates: %s", err)

		return nil, fmt.Errorf("failed to retrieve server certificates: %w", err)
	}

	clientCAs, err := x509.SystemCertPool()
	if err != nil {
		clientCAs = x509.NewCertPool()
	}

	for _, ce := range clientCerts {
		clientCAs.AppendCertsFromPEM([]byte(ce.Certificate))
	}

	tlsConfig := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		Certificates:     tlsCerts,
		ClientAuth:       tls.RequestClientCert, // client certs are manually verified
		ClientCAs:        clientCAs,
		VerifyConnection: compatibility.LogSha1(h.logger),
	}

	return tlsConfig, nil
}

func (h *httpService) listen(agent *model.LocalAgent) error {
	addr, err := conf.GetRealAddress(agent.Address)
	if err != nil {
		h.logger.Error("Failed to retrieve HTTP server address: %s", err)

		return fmt.Errorf("failed to retrieve HTTP server address: %w", err)
	}

	list, err := net.Listen("tcp", addr)
	if err != nil {
		h.logger.Error("Failed to start server listener: %s", err)

		return fmt.Errorf("failed to start server listener: %w", err)
	}

	go func() {
		var err error
		if agent.Protocol == "https" {
			err = h.serv.ServeTLS(list, "", "")
		} else {
			err = h.serv.Serve(list)
		}

		if !errors.Is(err, http.ErrServerClosed) {
			h.logger.Error("Unexpected error: %h", err)
			h.state.Set(state.Error, err.Error())
		} else {
			h.state.Set(state.Offline, "")
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

		var agent model.LocalAgent
		if err := h.db.Get(&agent, "id=?", h.agentID).Run(); err != nil {
			h.logger.Error("Failed to retrieve user credentials: %s", err)
			http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

			return
		}

		acc, canContinue := h.checkAuthent(w, r, &agent)
		if !canContinue {
			return
		}

		handler := &httpHandler{
			running: h.running,
			agent:   &agent,
			account: acc,
			db:      h.db,
			logger:  h.logger,
			req:     r,
			resp:    w,
		}

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

func (h *httpService) checkAuthent(w http.ResponseWriter, r *http.Request,
	agent *model.LocalAgent,
) (*model.LocalAccount, bool) {
	var acc *model.LocalAccount

	login, pswd, ok := r.BasicAuth()
	if !ok || login == "" {
		unauthorized(w, "auth: missing login")

		return nil, false
	}

	if pswd != "" {
		acc1, canContinue := h.passwdAuth(w, agent, login, pswd)
		if !canContinue {
			return nil, false
		}

		acc = acc1
	}

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		acc2, canContinue := h.certAuth(w, login, r.TLS.PeerCertificates, agent)
		if !canContinue {
			return nil, false
		}

		acc = acc2
	}

	if acc == nil {
		unauthorized(w, "missing credentials")

		return nil, false
	}

	return acc, true
}

func (h *httpService) passwdAuth(w http.ResponseWriter, agent *model.LocalAgent,
	login, pswd string,
) (*model.LocalAccount, bool) {
	var acc model.LocalAccount
	if err := h.db.Get(&acc, "login=? AND local_agent_id=?", login, agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			h.logger.Error("Failed to retrieve user credentials: %s", err)
			http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

			return nil, false
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(pswd)); err != nil {
		h.logger.Warning("Invalid credentials for user '%s'", login)
		unauthorized(w, "the given credentials are invalid")

		return nil, false
	}

	return &acc, true
}

func (h *httpService) certAuth(w http.ResponseWriter, login string, certs []*x509.Certificate,
	agent *model.LocalAgent,
) (*model.LocalAccount, bool) {
	var acc model.LocalAccount
	if err := h.db.Get(&acc, "login=? AND local_agent_id=?", login, agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			h.logger.Error("Failed to retrieve user credentials: %s", err)
			http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

			return nil, false
		}
	}

	var cryptos model.Cryptos
	if err := h.db.Select(&cryptos).Where("owner_type=? AND owner_id=?", acc.TableName(),
		acc.ID).Run(); err != nil {
		h.logger.Error("Failed to retrieve user crypto credentials: %s", err)
		http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

		return nil, false
	}

	if len(cryptos) == 0 {
		h.logger.Warning("No certificates found for user '%s'", login)
		unauthorized(w, "No certificates found for this user")

		return nil, false
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}

	for _, crypto := range cryptos {
		roots.AppendCertsFromPEM([]byte(crypto.Certificate))
	}

	intermediate := x509.NewCertPool()
	for _, cert := range certs {
		intermediate.AddCert(cert)
	}

	opt := x509.VerifyOptions{
		DNSName:       login,
		Roots:         roots,
		Intermediates: intermediate,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := certs[0].Verify(opt); err != nil {
		h.logger.Warning("Certificate is not valid for this user: %s", err)
		unauthorized(w, "Certificate is not valid for this user")

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
