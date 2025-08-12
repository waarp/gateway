package snmp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/lib/log"
	snmplib "github.com/slayercat/GoSNMPServer"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ServiceName = "SNMP"

//nolint:gochecknoglobals //global var is better for simplicity
var GlobalService *Service

type Service struct {
	DB        *database.DB
	Logger    *log.Logger
	state     utils.State
	startTime time.Time

	server *snmplib.SNMPServer

	monitors    []*MonitorConfig
	monConfLock sync.RWMutex
}

func (s *Service) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := s.start(); err != nil {
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.startTime = time.Now()

	return nil
}

func (s *Service) start() error {
	s.Logger = logging.NewLogger(ServiceName)
	s.Logger.Info("Starting service...")

	if err := s.startServer(); err != nil {
		return err
	}

	if err := s.ReloadMonitorsConf(); err != nil {
		return err
	}

	s.Logger.Info("Service started")

	return nil
}

func (s *Service) startServer() error {
	var serverConf ServerConfig
	if err := s.DB.Get(&serverConf, "owner=?", conf.GlobalConfig.GatewayName).
		Run(); database.IsNotFound(err) {
		s.server = nil

		return nil // no server configured
	} else if err != nil {
		s.Logger.Errorf("Failed to retrieve SNMP server configuration: %v", err)

		return fmt.Errorf("failed to retrieve SNMP server configuration: %w", err)
	}

	if err := s.listen(&serverConf); err != nil {
		s.Logger.Errorf("Failed to start SNMP server: %v", err)

		return fmt.Errorf("failed to start SNMP server: %w", err)
	}

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := s.stop(ctx); err != nil {
		s.state.Set(utils.StateError, err.Error())
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *Service) stop(ctx context.Context) error {
	s.Logger.Info("Stopping service...")

	return utils.RunWithCtx(ctx, func() error {
		if s.server != nil {
			s.server.Shutdown()
		}

		s.Logger.Info("Service stopped")

		return nil
	})
}

func (s *Service) State() (utils.StateCode, string) { return s.state.Get() }

func (s *Service) ReloadMonitorsConf() error {
	s.monConfLock.Lock()
	defer s.monConfLock.Unlock()

	var monitors model.Slice[*MonitorConfig]
	if err := s.DB.Select(&monitors).Run(); err != nil {
		s.Logger.Errorf("Failed to retrieve SNMP monitors: %v", err)

		return fmt.Errorf("failed to retrieve SNMP monitors: %w", err)
	}

	s.monitors = monitors

	return nil
}

func (s *Service) ReloadServerConf(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return err
	}

	return s.Start()
}

func (s *Service) ReportTransferError(transferID int64) {
	var trans model.NormalizedTransferView
	if err := s.DB.Get(&trans, "id = ?", transferID).Run(); err != nil {
		s.Logger.Errorf("Failed to retrieve transfer: %v", err)

		return
	}

	if err := s.sendTransferError(&trans); err != nil {
		s.Logger.Errorf("Failed to send transfer error: %v", err)
	}
}

func (s *Service) SendTestNotification() error {
	trans := &model.NormalizedTransferView{
		HistoryEntry: model.HistoryEntry{
			ID:               -1,
			RemoteTransferID: "",
			IsServer:         false,
			IsSend:           true,
			Rule:             "test_rule",
			Account:          "test_account",
			Agent:            "test_agent",
			Client:           "test_client",
			LocalPath:        "/test/file",
			ErrCode:          types.TeInternal,
			ErrDetails:       "This is a test notification",
		},
		IsTransfer: true,
	}

	return s.sendTransferError(trans)
}

func ReportServiceFailure(service string, sErr error) {
	if GlobalService == nil {
		return
	}

	if err := GlobalService.sendServiceError(service, sErr); err != nil {
		GlobalService.Logger.Errorf("Failed to send service error: %v", err)
	}
}

// sysUpTime returns the elapsed time since the service started (in hundredths
// of a second).
func (s *Service) sysUpTime() uint32 {
	return timeTicksSince(s.startTime)
}

func (s *Service) GetServerAddr() string {
	if s.server == nil {
		return ""
	}

	addr := s.server.Address()
	if addr == nil {
		return ""
	}

	return addr.String()
}

func (s *Service) StartNoServer() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := s.startNoServer(); err != nil {
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.startTime = time.Now()

	return nil
}

func (s *Service) startNoServer() error {
	s.Logger = logging.NewLogger(ServiceName)
	s.Logger.Info("Starting service...")

	if err := s.ReloadMonitorsConf(); err != nil {
		return err
	}

	s.Logger.Info("Service started")

	return nil
}
