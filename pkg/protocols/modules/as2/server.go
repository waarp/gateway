package as2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"code.waarp.fr/lib/as2"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type server struct {
	db     *database.DB
	logger *log.Logger
	state  utils.State

	agent    *model.LocalAgent
	listener net.Listener
	server   *as2.Server
	tracer   func() pipeline.Trace
}

func NewServer(db *database.DB, agent *model.LocalAgent) protocol.Server {
	return &server{db: db, agent: agent}
}

func (s *server) Name() string                     { return s.agent.Name }
func (s *server) State() (utils.StateCode, string) { return s.state.Get() }

func (s *server) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.agent.Name)
	s.logger.Info("Starting AS2 server...")

	if err := s.start(); err != nil {
		s.logger.Errorf("Failed to start AS2 service: %v", err)
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")
	s.logger.Infof("AS2 server started successfully on %q", s.listener.Addr().String())

	return nil
}

func (s *server) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.logger.Info("Shutting down AS2 server")

	if err := s.stop(ctx); err != nil {
		s.logger.Error(err.Error())
		_ = s.listener.Close() //nolint:errcheck //error is irrelevant at this point
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")
	s.logger.Info("AS2 server shutdown successful")

	return nil
}

func (s *server) start() error {
	var servConf serverProtoConfigTLS
	if err := utils.JSONConvert(s.agent.ProtoConfig, &servConf); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	realAddr := conf.GetRealAddress(s.agent.Address.Host,
		utils.FormatUint(s.agent.Address.Port),
	)

	var listErr error
	if s.agent.Protocol == AS2TLS {
		s.listener, listErr = tls.Listen("tcp", realAddr, &tls.Config{
			MinVersion:         servConf.MinTLSVersion.TLS(),
			GetConfigForClient: protoutils.GetServerTLSConfig(s.db, s.logger, s.agent, servConf.MinTLSVersion),
		})
	} else {
		s.listener, listErr = net.Listen("tcp", realAddr)
	}

	if listErr != nil {
		return fmt.Errorf("failed to start server listener: %w", listErr)
	}

	s.server = as2.NewServer(
		as2.WithLocalAS2ID(s.agent.Name),
		as2.WithInboundPath("/"),
		as2.WithMaxBodyBytes(int64(servConf.MaxFileSize)),
		as2.WithLogger(slogs(s.logger)),
		as2.WithPartnerStore(s.newPartnerStore()),
		as2.WithHTTPAuthFunc(s.auth),
		as2.WithFileHandler(s.handle),
		as2.WithMDNSignDigest(servConf.MDNSignatureAlgorithm.as2()),
		func(server *as2.Server) {
			creds, dbErr := s.agent.GetCredentials(s.db, auth.TLSCertificate)
			if dbErr != nil {
				s.logger.Errorf("Failed to get credentials: %v", dbErr)

				return
			}

			for _, cred := range creds {
				cert, err := tls.X509KeyPair([]byte(cred.Value), []byte(cred.Value2))
				if err != nil {
					s.logger.Warningf("Failed to parse TLS certificate %q: %v", cred.Name, err)
					continue
				}

				as2.WithMDNSigner(cert.Leaf, cert.PrivateKey)(server)
				as2.WithDecryptCertKey(cert.Leaf, cert.PrivateKey)(server)

				return
			}
		},
	)

	//nolint:errcheck //we don't care about the error here
	go s.server.Serve(s.listener)

	return nil
}

func (s *server) stop(ctx context.Context) error {
	res := utils.GoRun(func() error {
		if err := pipeline.List.StopAllFromServer(ctx, s.agent.ID); err != nil {
			s.logger.Errorf("Failed to interrupt AS2 transfers: %v", err)

			return fmt.Errorf("could not halt the service gracefully: %w", err)
		}

		return nil
	})

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Errorf("Failed to shutdown AS2 server: %v", err)

		return fmt.Errorf("failed to close listener: %w", err)
	}

	return <-res
}

func (s *server) SetTracer(tracer func() pipeline.Trace) {
	s.tracer = tracer
}
