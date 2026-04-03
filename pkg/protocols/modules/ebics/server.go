package ebics

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	liborders "code.waarp.fr/lib/ebics/ebics/orders"
	libadminhelper "code.waarp.fr/lib/ebics/ebics/provider/adminhelper"
	libproviderserver "code.waarp.fr/lib/ebics/ebics/provider/server"
	libserver "code.waarp.fr/lib/ebics/ebics/server"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// Server hosts the lib-ebics HTTP handler inside the Gateway service lifecycle.
type Server struct {
	db     *database.DB
	logger *log.Logger
	server *model.LocalAgent
	config *serverConfig
	state  utils.State

	providerStore *providerStore
	ebicsServer   *libserver.Server
	httpServer    *http.Server
	listener      net.Listener
}

// NewServer returns the EBICS server instance bound to a Gateway local agent.
func NewServer(db *database.DB, dbServer *model.LocalAgent) *Server {
	return &Server{db: db, server: dbServer}
}

// Start boots the Gateway EBICS service and the underlying lib-ebics handler.
func (s *Server) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.server.Name)
	s.logger.Info("Starting EBICS server...")

	if err := s.start(); err != nil {
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.server.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Infof("EBICS server started successfully on %q", s.httpServer.Addr)

	return nil
}

func (s *Server) start() error {
	cfg := defaultServerConfig()
	if err := utils.JSONConvert(s.server.ProtoConfig, cfg); err != nil {
		return wrapConfigError(err)
	}

	if err := cfg.ValidServer(); err != nil {
		return wrapConfigError(err)
	}

	s.config = cfg
	s.providerStore = newProviderStore(s.db)

	adminHandler, err := libadminhelper.NewWithDefaultKeyMgmt(s.providerStore, nil, newServerAdminPolicy(s.providerStore))
	if err != nil {
		return fmt.Errorf("initialize lib-ebics admin helper: %w", err)
	}

	router := libadminhelper.NewRouter(adminHandler)
	payloadRouter := newPayloadOrderRouter(s.db, s.logger)
	hpdRouter := newContractOrderProvider(s.db, s.providerStore, s.server, "HPD")
	hkdRouter := newContractOrderProvider(s.db, s.providerStore, s.server, "HKD")
	htdRouter := newContractOrderProvider(s.db, s.providerStore, s.server, "HTD")
	haaRouter := newContractOrderProvider(s.db, s.providerStore, s.server, "HAA")
	hvdRouter := newServerReportingProvider(s.db, s.providerStore, "HVD")
	hveRouter := newServerReportingProvider(s.db, s.providerStore, "HVE")
	hvuRouter := newServerReportingProvider(s.db, s.providerStore, "HVU")
	hvzRouter := newServerReportingProvider(s.db, s.providerStore, "HVZ")
	hvtRouter := newServerReportingProvider(s.db, s.providerStore, "HVT")
	hacRouter := newServerReportingProvider(s.db, s.providerStore, "HAC")
	hvsRouter := newServerReportingProvider(s.db, s.providerStore, "HVS")
	router.Register("FUL", liborders.FULHandler{Provider: payloadRouter})
	router.Register("FDL", liborders.FDLHandler{Provider: payloadRouter})
	router.Register("BTU", liborders.BTUHandler{Provider: payloadRouter})
	router.Register("BTD", liborders.BTDHandler{Provider: payloadRouter})
	router.Register("HPD", liborders.HPDHandler{Provider: hpdRouter})
	router.Register("HKD", liborders.HKDHandler{Provider: hkdRouter})
	router.Register("HTD", liborders.HTDHandler{Provider: htdRouter})
	router.Register("HAA", liborders.HAAHandler{Provider: haaRouter})
	router.Register("HVD", liborders.HVDHandler{Provider: hvdRouter})
	router.Register("HVE", hveHandler{provider: hveRouter})
	router.Register("HVU", liborders.HVUHandler{Provider: hvuProviderAdapter{hvuRouter}})
	router.Register("HVZ", liborders.HVZHandler{Provider: hvzProviderAdapter{hvzRouter}})
	router.Register("HVS", liborders.HVSHandler{Provider: hvsRouter})
	router.Register("HVT", liborders.HVTHandler{
		Provider:             hvtProviderAdapter{hvtRouter},
		OriginalDataProvider: hvtRouter,
	})
	router.Register("HAC", liborders.HACHandler{DocumentProvider: hacRouter})

	builder := libproviderserver.New().
		KeyStore(s.providerStore).
		SubscriberStore(s.providerStore).
		OrderHandler(router).
		Observer(newServerObserver(s.logger)).
		TxStore(s.providerStore).
		NonceStore(s.providerStore).
		Logger(s.logger.AsStdLogger(log.LevelInfo)).
		RequireTLS(true).
		RequireMTLS(false).
		RequireCorrelationID(false).
		RequireTenantID(false).
		HandlerTimeout(time.Duration(cfg.RequestTimeout) * time.Second).
		Option(libserver.WithSigner(newProviderRequestSigner(s.providerStore))).
		Option(libserver.WithVerifyBankDigests(cfg.VerifyBankKeys)).
		Option(libserver.WithMaxSegmentBytes(cfg.MaxSegmentSize)).
		Option(libserver.WithEDSReferenceDataProvider(hveRouter)).
		Option(libserver.WithRequireTxStore(true))

	if xsdDir, ok := resolveEBICSXSDDir(); ok {
		builder.StrictH005XSDProfile(xsdDir)
	} else {
		builder.DisableStrictXSDDefault()
		s.logger.Warning("EBICS strict XSD profile disabled: no XSD base directory was found")
	}

	ebicsServer, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build lib-ebics provider server: %w", err)
	}

	s.ebicsServer = ebicsServer
	s.httpServer = &http.Server{
		Addr:              conf.GetRealAddress(s.server.Address.Host, utils.FormatUint(s.server.Address.Port)),
		Handler:           ebicsServer.Handler(),
		ErrorLog:          s.logger.AsStdLogger(log.LevelWarning),
		ReadHeaderTimeout: time.Duration(cfg.RequestTimeout) * time.Second,
		ConnState: func(_ net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				analytics.AddIncomingConnection()
			case http.StateClosed:
				analytics.SubIncomingConnection()
			default:
			}
		},
	}

	listener, err := s.listen()
	if err != nil {
		return err
	}

	s.listener = listener
	httpServer := s.httpServer

	go func(server *http.Server, serveListener net.Listener) {
		serveErr := server.Serve(serveListener)
		if !errors.Is(serveErr, http.ErrServerClosed) {
			s.logger.Errorf("Unexpected EBICS server error: %v", serveErr)
			s.state.Set(utils.StateError, fmt.Sprintf("unexpected error: %v", serveErr))
		} else {
			s.state.Set(utils.StateOffline, "")
		}
	}(httpServer, listener)

	return nil
}

func (s *Server) listen() (net.Listener, error) {
	tlsConfig := &tls.Config{
		MinVersion:         s.config.MinTLSVersion.TLS(),
		GetConfigForClient: protoutils.GetServerTLSConfig(s.db, s.logger, s.server, s.config.MinTLSVersion),
	}

	listener, err := tls.Listen("tcp", s.httpServer.Addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("start EBICS TLS listener: %w", err)
	}

	return listener, nil
}

// Stop gracefully shuts down the HTTP listener hosting the lib-ebics server.
func (s *Server) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Shutting down EBICS server...")

	if err := s.stop(ctx); err != nil {
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.server.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Server) stop(ctx context.Context) error {
	if err := pipeline.List.StopAllFromServer(ctx, s.server.ID); err != nil {
		return fmt.Errorf("interrupt active Gateway pipelines for EBICS server: %w", err)
	}

	if err := s.httpServer.Shutdown(ctx); err != nil {
		if closeErr := s.httpServer.Close(); closeErr != nil {
			s.logger.Warningf("forced EBICS server close also failed: %v", closeErr)
		}

		return fmt.Errorf("shutdown EBICS HTTP server: %w", err)
	}

	s.listener = nil
	s.httpServer = nil
	s.ebicsServer = nil
	s.providerStore = nil
	s.config = nil

	return nil
}

// State returns the current Gateway service state for the EBICS server.
func (s *Server) State() (utils.StateCode, string) {
	return s.state.Get()
}

func resolveEBICSXSDDir() (string, bool) {
	if fromEnv := strings.TrimSpace(os.Getenv("EBICS_XSD_DIR")); fromEnv != "" {
		if dirExists(fromEnv) {
			return fromEnv, true
		}
	}

	candidates := []string{
		filepath.Clean(filepath.Join("..", "EBICS", "internal", "assets", "xsd")),
		filepath.Clean(`C:\MonProjet\EBICS\internal\assets\xsd`),
	}

	for _, candidate := range candidates {
		if dirExists(candidate) {
			return candidate, true
		}
	}

	return "", false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
