package sftp

import (
	"context"
	"fmt"

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

func (s *service) Name() string { return s.server.Name }

func (s *service) reportError(err error) {
	if err == nil {
		return
	}

	s.logger.Error(err.Error())
	s.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(s.server.Name, err)
}

func (s *service) start() error {
	addr := s.db.Config.Overrides.GetRealAddress(s.server.Address.Host,
		utils.FormatUint(s.server.Address.Port))

	listener, err3 := protoutils.Listen("tcp", addr)
	if err3 != nil {
		return fmt.Errorf("failed to start server listener: %w", err3)
	}

	s.listener = &sshListener{
		DB:       s.db,
		Logger:   s.logger,
		serverID: s.server.ID,
		listener: listener,
		shutdown: make(chan struct{}),
	}

	if n, err := s.db.Count(&model.Credential{}).
		Where("local_agent_id=?", s.server.ID).
		Where("type=?", AuthSSHPrivateKey).Run(); err == nil && n == 0 {
		s.logger.Warning("Server has no hostkey and will not be accessible without one")
	}

	go s.listener.listen()

	return nil
}

// Start starts the SFTP service.
func (s *service) Start() (retErr error) {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.server.Name)
	defer s.reportError(retErr)

	if err := s.db.Get(s.server, "id=?", s.server.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the SFTP server: %w", err)
	}

	s.logger = logging.NewLogger(s.server.Name)
	s.logger.Info("Starting SFTP server...")

	if err := s.start(); err != nil {
		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Infof("SFTP server started successfully on %q", s.listener.listener.Addr().String())

	return nil
}

// Stop stops the SFTP service.
func (s *service) Stop(ctx context.Context) (retErr error) {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Shutting down SFTP server")
	defer s.reportError(retErr)

	if err := s.listener.close(ctx); err != nil {
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
