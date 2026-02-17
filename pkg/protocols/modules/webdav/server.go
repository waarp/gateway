package webdav

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"
	"golang.org/x/net/webdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const readHeaderTimeout = 10 * time.Second

type server struct {
	db     *database.DB
	logger *log.Logger

	state  utils.State
	tracer func() pipeline.Trace

	agent  *model.LocalAgent
	conf   *ServerConfigTLS
	server *http.Server
	lock   webdav.LockSystem
}

func NewServer(db *database.DB, dbServer *model.LocalAgent) services.Server {
	return &server{db: db, agent: dbServer}
}

func (s *server) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.agent.Name)
	s.logger.Info("Starting Webdav server...")

	if err := s.start(); err != nil {
		s.logger.Errorf("Failed to start Webdav service: %v", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Infof("Webdav server started successfully on %q", s.server.Addr)

	return nil
}

func (s *server) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Shutting down Webdav server")

	if err := s.stop(ctx); err != nil {
		s.logger.Error(err.Error())
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")
	s.logger.Info("SFTP server shutdown successful")

	return nil
}

func (s *server) State() (utils.StateCode, string) {
	return s.state.Get()
}
func (s *server) SetTracer(tracer func() pipeline.Trace) { s.tracer = tracer }

func (s *server) start() error {
	if err := utils.JSONConvert(s.agent.ProtoConfig, &s.conf); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	s.lock = webdav.NewMemLS()
	addr := conf.GetRealAddress(s.agent.Address.Host,
		utils.FormatUint(s.agent.Address.Port))

	s.server = &http.Server{
		Addr:              addr,
		Handler:           http.HandlerFunc(s.handle),
		ErrorLog:          s.logger.AsStdLogger(log.LevelWarning),
		ReadHeaderTimeout: readHeaderTimeout,
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

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server listener: %w", err)
	}

	if s.agent.Protocol == WebdavTLS {
		listener = s.listentTLS(listener)
	}

	//nolint:errcheck //error does not matter here
	go s.server.Serve(listener)

	return nil
}

func (s *server) stop(ctx context.Context) error {
	stopErr := make(chan error, 1)
	go s.interruptTransfers(ctx, stopErr)

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shutdown server, forcing exit...")
		_ = s.server.Close()

		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return <-stopErr
}

func (s *server) interruptTransfers(ctx context.Context, res chan<- error) {
	defer close(res)

	if err := pipeline.List.StopAllFromServer(ctx, s.agent.ID); err != nil {
		s.logger.Errorf("Failed to interrupt Webdav transfers: %v", err)

		res <- fmt.Errorf("could not halt the service gracefully: %w", err)
	}
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	s.logger.Debugf("Webdav %s request received on %q", r.Method, r.URL.String())
	account, ok := s.auth(w, r)
	if !ok {
		return
	}

	handler := &webdav.Handler{
		Prefix: "",
		FileSystem: &webdavFS{
			db:      s.db,
			logger:  s.logger,
			req:     r,
			tracer:  s.tracer,
			server:  s.agent,
			account: account,
		},
		LockSystem: s.lock,
		Logger: func(r *http.Request, err error) {
			if err != nil {
				s.logger.Errorf("Webdav %s request to %q failed: %v",
					r.Method, r.URL.String(), err)
			}
		},
	}

	handler.ServeHTTP(w, r)
	s.logger.Debugf("Webdav %s request processed", r.Method)
}

func (s *server) listentTLS(l net.Listener) net.Listener {
	tlsConfig := &tls.Config{
		MinVersion:         s.conf.MinTLSVersion.TLS(),
		GetConfigForClient: protoutils.GetServerTLSConfig(s.db, s.logger, s.agent, s.conf.MinTLSVersion),
	}

	return tls.NewListener(l, tlsConfig)
}
