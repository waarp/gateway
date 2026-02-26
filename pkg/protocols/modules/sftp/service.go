package sftp

import (
	"context"
	"fmt"
	"net"

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

// service represents an instance of SFTP server.
type service struct {
	db     *database.DB
	server *model.LocalAgent

	state    utils.State
	logger   *log.Logger
	listener *sshListener
}

func (s *service) start() error {
	var protoConfig serverConfig
	if err := utils.JSONConvert(s.server.ProtoConfig, &protoConfig); err != nil {
		return fmt.Errorf("cannot parse the protocol configuration of this agent: %w", err)
	}

	hostKeys, err := s.server.GetCredentials(s.db, AuthSSHPrivateKey)
	if err != nil {
		s.logger.Errorf("Failed to retrieve the server host keys: %v", err)

		return fmt.Errorf("failed to retrieve the server host keys: %w", err)
	}

	sshConf, err1 := getSSHServerConfig(s.db, s.logger, hostKeys, &protoConfig, s.server)
	if err1 != nil {
		s.logger.Errorf("Failed to parse the SSH server configuration: %v", err1)

		return fmt.Errorf("failed to parse the SSH server configuration: %w", err1)
	}

	addr := conf.GetRealAddress(s.server.Address.Host,
		utils.FormatUint(s.server.Address.Port))

	listener, err3 := net.Listen("tcp", addr)
	if err3 != nil {
		s.logger.Errorf("Failed to start server listener: %v", err3)

		return fmt.Errorf("failed to start server listener: %w", err3)
	}

	listener = &protoutils.TraceListener{Listener: listener}

	s.listener = &sshListener{
		DB:       s.db,
		Logger:   s.logger,
		Server:   s.server,
		SSHConf:  sshConf,
		Listener: listener,
		shutdown: make(chan struct{}),
	}

	go s.listener.listen()

	return nil
}

// Start starts the SFTP service.
func (s *service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.server.Name)
	s.logger.Info("Starting SFTP server...")

	if err := s.start(); err != nil {
		s.logger.Errorf("Failed to start SFTP service: %v", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.server.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Infof("SFTP server started successfully on %q", s.listener.Listener.Addr().String())

	return nil
}

// Stop stops the SFTP service.
func (s *service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	defer func() { s.listener = nil }()

	s.logger.Info("Shutting down SFTP server")

	if err := s.listener.close(ctx); err != nil {
		s.logger.Error("Failed to shut down SFTP server, forcing exit")
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.server.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")
	s.logger.Info("SFTP server shutdown successful")

	return nil
}

func (s *service) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *service) SetTracer(getTrace func() pipeline.Trace) {
	s.listener.tracer = getTrace
}
