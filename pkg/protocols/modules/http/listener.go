package http

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func (h *httpService) listen() error {
	addr := h.db.Config.Overrides.GetRealAddress(h.agent.Address.Host,
		utils.FormatUint(h.agent.Address.Port))
	h.serv.Addr = addr

	var (
		list   net.Listener
		netErr error
	)

	if h.agent.Protocol == HTTPS {
		tlsConfig := protoutils.GetServerTLSConfig(h.db, h.logger, h.agent.ID)
		list, netErr = protoutils.ListenTLS("tcp", addr, tlsConfig)
	} else {
		list, netErr = protoutils.Listen("tcp", addr)
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
	var authenticated bool

	login, pswd, ok := r.BasicAuth()
	if !ok || login == "" {
		unauthorized(w, "auth: missing login")

		return nil, false
	}

	// We purposefully ignore NotFound errors to avoid leaking information
	// about the existence of an account.
	acc, accErr := h.agent.GetAccount(h.db, login)
	if accErr != nil && !database.IsNotFound(accErr) {
		h.logger.Errorf("Failed to retrieve user credentials: %v", accErr)
		http.Error(w, "Failed to retrieve user credentials", http.StatusInternalServerError)

		return nil, false
	}

	if success, err := protoutils.CertificateAuthentication(h.db, h.logger, acc, r.TLS); err != nil {
		http.Error(w, "internal authentication error", http.StatusInternalServerError)

		return nil, false
	} else if success {
		authenticated = true
	}

	if success, err := protoutils.PasswordAuthentication(h.db, h.logger, acc, pswd); err != nil {
		http.Error(w, "internal authentication error", http.StatusInternalServerError)

		return nil, false
	} else if success {
		authenticated = true
	}

	if !authenticated {
		unauthorized(w, "auth: invalid authentication")

		return nil, false
	}

	if !acc.CheckIP(h.logger, r.RemoteAddr) {
		unauthorized(w, "auth: unauthorized IP address")

		return nil, false
	}

	return acc, true
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
