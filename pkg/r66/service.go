package r66

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// Service represents a r66 service, which encompasses a r66 server usable for
// transfers.
type Service struct {
	db     *database.DB
	logger *log.Logger
	agent  *model.LocalAgent

	cancel context.CancelFunc
	paths  pipeline.Paths

	state  *service.State
	server *r66.Server

	done chan struct{}
}

// NewService returns a new R66 service instance with the given attributes.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) *Service {
	paths := pipeline.Paths{
		PathsConfig: db.Conf.Paths,
		ServerRoot:  agent.Root,
		ServerIn:    agent.InDir,
		ServerOut:   agent.OutDir,
		ServerWork:  agent.WorkDir,
	}

	return &Service{
		db:     db,
		agent:  agent,
		logger: logger,
		paths:  paths,
		state:  &service.State{},
	}
}

func (s *Service) makeTLSConf() (*tls.Config, error) {
	certs, err := s.agent.GetCryptos(s.db)
	if err != nil {
		return nil, err
	}

	if len(certs) == 0 {
		return nil, nil
	}

	tlsCerts := make([]tls.Certificate, len(certs))

	for i := range certs {
		var err error
		tlsCerts[i], err = tls.X509KeyPair(
			[]byte(certs[i].Certificate),
			[]byte(certs[i].PrivateKey),
		)

		if err != nil {
			return nil, fmt.Errorf("cannot generate certificate list: %w", err)
		}
	}

	// ca, _ := x509.SystemCertPool()
	return &tls.Config{
		Certificates: tlsCerts,
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		// ClientCAs:    ca,
	}, nil
}

// Start launches a r66 service with an integrated r66 server.
func (s *Service) Start() error {
	s.logger.Infof("Starting R66 server '%s'...", s.agent.Name)
	s.state.Set(service.Starting, "")

	var conf config.R66ProtoConfig
	if err := json.Unmarshal(s.agent.ProtoConfig, &conf); err != nil {
		s.logger.Errorf("Failed to parse server ProtoConfig: %s", err)
		err1 := fmt.Errorf("failed to parse ProtoConfig: %w", err)
		s.state.Set(service.Error, err1.Error())

		return err1
	}

	pwd, err := utils.AESDecrypt(conf.ServerPassword)
	if err != nil {
		s.logger.Errorf("Failed to decrypt server password: %s", err)
		dErr := fmt.Errorf("failed to decrypt server password: %w", err)
		s.state.Set(service.Error, dErr.Error())

		return dErr
	}

	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())

	s.server = &r66.Server{
		Login:    s.agent.Name,
		Password: []byte(pwd),
		Conf: &r66.Config{
			FileSize:  true,
			FinalHash: true,
			HashAlgo:  "SHA-256",
			Proxified: false,
		},
		Handler: &authHandler{
			Service: s,
			ctx:     ctx,
		},
		Logger: s.logger.AsStdLog(logging.DEBUG),
	}

	if err := s.listen(); err != nil {
		return err
	}

	s.state.Set(service.Running, "")
	s.logger.Infof("R66 server started at %s", s.agent.Address)

	return nil
}

func (s *Service) listen() error {
	tlsConf, err := s.makeTLSConf()
	if err != nil {
		s.logger.Errorf("Failed to parse server TLS config: %s", err)
		s.state.Set(service.Error, err.Error())

		return err
	}

	var list net.Listener
	if tlsConf != nil {
		list, err = tls.Listen("tcp", s.agent.Address, tlsConf)
	} else {
		list, err = net.Listen("tcp", s.agent.Address)
	}

	if err != nil {
		s.logger.Errorf("Failed to start R66 listener: %s", err)
		s.state.Set(service.Error, err.Error())

		return fmt.Errorf("cannot start listener: %w", err)
	}

	s.done = make(chan struct{})

	go func() {
		if err := s.server.Serve(list); err != nil {
			s.logger.Errorf("Server has stopped unexpectedly: %s", err)
			s.state.Set(service.Error, err.Error())
		}

		close(s.done)
	}()

	return nil
}

// Stop shuts down the r66 server and stops the service.
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Infof("Shutting down R66 server")

	if code, _ := s.State().Get(); code == service.Error || code == service.Offline {
		s.logger.Info("Server is already offline, nothing to do")

		return nil
	}

	s.state.Set(service.ShuttingDown, "")
	defer s.state.Set(service.Offline, "")

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shut down R66 server, forcing exit")

		return fmt.Errorf("failed to shut down R66 server: %w", err)
	}

	s.logger.Info("R66 server shutdown successful")

	select {
	case <-ctx.Done():
		s.logger.Error("Failed to shut down R66 server, forcing exit")

		if ctx.Err() != nil {
			return fmt.Errorf("failed to shut down R66 server: %w", ctx.Err())
		}

		return nil
	case <-s.done:
	}

	return nil
}

// State returns the r66 service's state.
func (s *Service) State() *service.State {
	if s.server == nil {
		if code, _ := s.state.Get(); code == service.Running {
			s.state.Set(service.Offline, "")
		}
	}

	return s.state
}
