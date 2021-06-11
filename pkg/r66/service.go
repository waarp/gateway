// Package r66 contains the functions necessary to execute a file transfer
// using the R66 protocol. The package defines both a client and a server.
package r66

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-r66/r66"
)

func init() {
	gatewayd.ServiceConstructors["r66"] = NewService
}

// Service represents a r66 service, which encompasses a r66 server usable for
// transfers.
type Service struct {
	db     *database.DB
	logger *log.Logger
	agent  *model.LocalAgent
	state  *service.State

	server *r66.Server

	runningTransfers *pipeline.TransferMap
}

// NewService returns a new R66 service instance with the given attributes.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.Service {
	return &Service{
		db:               db,
		agent:            agent,
		logger:           logger,
		state:            &service.State{},
		runningTransfers: pipeline.NewTransferMap(),
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
	for i, cert := range certs {
		var err error
		tlsCerts[i], err = tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
		if err != nil {
			return nil, err
		}
	}

	ca, cErr := x509.SystemCertPool()
	if cErr != nil {
		ca = x509.NewCertPool()
	}
	var clientCerts model.Cryptos
	if err := s.db.Select(&clientCerts).Where(
		"owner_type=? AND owner_id IN (SELECT id FROM local_accounts WHERE local_agent_id=?)",
		"local_accounts", s.agent.ID).Run(); err != nil {
		return nil, err
	}
	for _, cert := range clientCerts {
		if !ca.AppendCertsFromPEM([]byte(cert.Certificate)) {
			return nil, fmt.Errorf("failed to add cert %s to trusted certificates pool", cert.Name)
		}
	}

	return &tls.Config{
		Certificates: tlsCerts,
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    ca,
	}, nil
}

// Start launches a r66 service with an integrated r66 server.
func (s *Service) Start() error {
	s.logger.Infof("Starting R66 server '%s'...", s.agent.Name)
	s.state.Set(service.Starting, "")

	var conf config.R66ProtoConfig
	if err := json.Unmarshal(s.agent.ProtoConfig, &conf); err != nil {
		s.logger.Errorf("Failed to parse server ProtoConfig: %s", err)
		err1 := fmt.Errorf("failed to parse ProtoConfig: %s", err)
		s.state.Set(service.Error, err1.Error())
		return err1
	}

	pwd, err := utils.AESDecrypt(database.GCM, conf.ServerPassword)
	if err != nil {
		s.logger.Errorf("Failed to decrypt server password: %s", err)
		dErr := fmt.Errorf("failed to decrypt server password: %s", err)
		s.state.Set(service.Error, dErr.Error())
		return dErr
	}

	s.server = &r66.Server{
		Login:    conf.ServerLogin,
		Password: []byte(pwd),
		Logger:   s.logger.AsStdLog(logging.DEBUG),
		Conf: &r66.Config{
			FileSize:   true,
			FinalHash:  true,
			DigestAlgo: "SHA-256",
			Proxified:  false,
		},
		Handler: &authHandler{Service: s},
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
		s.state.Set(service.Error, "failed to parse server TLS config")
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
		return err
	}

	go func() {
		if err := s.server.Serve(list); err != nil {
			s.logger.Errorf("Server has stopped unexpectedly: %s", err)
			s.state.Set(service.Error, err.Error())
		}
		_ = list.Close()
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

	go s.runningTransfers.InterruptAll()
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shut down R66 server, forcing exit")
		return err
	}
	s.logger.Info("R66 server shutdown successful")
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
