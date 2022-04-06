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
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errNoValidCert = errors.New("could not find a valid certificate for HTTP server")

func (h *httpService) makeTLSConf() (*tls.Config, error) {
	var certs model.Cryptos
	if err := h.db.Select(&certs).Where("owner_type=? AND owner_id=?", h.agent.TableName(),
		h.agent.ID).Run(); err != nil {
		h.logger.Errorf("Failed to retrieve server certificates: %s", err)

		return nil, fmt.Errorf("failed to retrieve server certificates: %w", err)
	}

	var tlsCerts []tls.Certificate

	for _, c := range certs {
		cert, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey))
		if err != nil {
			h.logger.Warningf("Failed to parse server certificate: %s", err)

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
		h.agent.ID).Run(); err != nil {
		h.logger.Errorf("Failed to retrieve client certificates: %s", err)

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

func (h *httpService) listen() error {
	addr, err := conf.GetRealAddress(h.agent.Address)
	if err != nil {
		h.logger.Errorf("Failed to retrieve HTTP server address: %s", err)

		return fmt.Errorf("failed to retrieve HTTP server address: %w", err)
	}

	list, err := net.Listen("tcp", addr)
	if err != nil {
		h.logger.Errorf("Failed to start server listener: %s", err)

		return fmt.Errorf("failed to start server listener: %w", err)
	}

	go func() {
		var err error
		if h.agent.Protocol == "https" {
			err = h.serv.ServeTLS(list, "", "")
		} else {
			err = h.serv.Serve(list)
		}

		if !errors.Is(err, http.ErrServerClosed) {
			h.logger.Errorf("Unexpected error: %h", err)
			h.state.Set(service.Error, err.Error())
		} else {
			h.state.Set(service.Offline, "")
		}
	}()

	return nil
}

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
			running: h.running,
			agent:   h.agent,
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
) (*model.LocalAccount, bool) {
	var acc *model.LocalAccount

	login, pswd, ok := r.BasicAuth()
	if !ok || login == "" {
		unauthorized(w, "auth: missing login")

		return nil, false
	}

	if pswd != "" {
		acc1, canContinue := h.passwdAuth(w, login, pswd)
		if !canContinue {
			return nil, false
		}

		acc = acc1
	}

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		acc2, canContinue := h.certAuth(w, login, r.TLS.PeerCertificates)
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

func (h *httpService) passwdAuth(w http.ResponseWriter, login, pswd string,
) (*model.LocalAccount, bool) {
	var acc model.LocalAccount
	if err := h.db.Get(&acc, "login=? AND local_agent_id=?", login, h.agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			h.logger.Errorf("Failed to retrieve user credentials: %s", err)
			http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

			return nil, false
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(pswd)); err != nil {
		h.logger.Warningf("Invalid credentials for user '%s'", login)
		unauthorized(w, "the given credentials are invalid")

		return nil, false
	}

	return &acc, true
}

func (h *httpService) certAuth(w http.ResponseWriter, login string, certs []*x509.Certificate,
) (*model.LocalAccount, bool) {
	var acc model.LocalAccount
	if err := h.db.Get(&acc, "login=? AND local_agent_id=?", login, h.agent.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			h.logger.Errorf("Failed to retrieve user credentials: %s", err)
			http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

			return nil, false
		}
	}

	var cryptos model.Cryptos
	if err := h.db.Select(&cryptos).Where("owner_type=? AND owner_id=?", acc.TableName(),
		acc.ID).Run(); err != nil {
		h.logger.Errorf("Failed to retrieve user crypto credentials: %s", err)
		http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

		return nil, false
	}

	if len(cryptos) == 0 {
		h.logger.Warningf("No certificates found for user '%s'", login)
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
		h.logger.Warningf("Certificate is not valid for this user: %s", err)
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
