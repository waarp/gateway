package pesit

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type server struct {
	db     *database.DB
	logger *log.Logger
	tracer func() pipeline.Trace
	server *pesit.Server
	state  utils.State

	localAgent *model.LocalAgent
	conf       *ServerConfigTLS
}

func (s *server) listen() (string, error) {
	s.server = pesit.NewServer(s)
	s.server.Logger = s.logger.AsStdLogger(log.LevelDebug)
	realAddr := conf.GetRealAddress(s.localAgent.Address.Host,
		utils.FormatUint(s.localAgent.Address.Port))

	var (
		list    net.Listener
		listErr error
	)

	if s.localAgent.Protocol == PesitTLS {
		tlsConfig := &tls.Config{
			MinVersion:            s.conf.MinTLSVersion.TLS(),
			GetCertificate:        s.getCertificate,
			VerifyPeerCertificate: auth.VerifyClientCert(s.db, s.logger, s.localAgent),
		}

		list, listErr = tls.Listen("tcp", realAddr, tlsConfig)
	} else {
		list, listErr = net.Listen("tcp", realAddr)
	}

	if listErr != nil {
		return "", fmt.Errorf("failed to open listener: %w", listErr)
	}

	list = &protoutils.TraceListener{Listener: list}

	s.server.SetPreConnectionUsage(s.conf.DisablePreConnection)

	go func() {
		if err := s.server.Serve(list); err != nil {
			s.logger.Errorf("unexpected error: %v", err)
			s.state.Set(utils.StateError, err.Error())
		}
	}()

	return list.Addr().String(), nil
}

func (s *server) stop(ctx context.Context) error {
	if err := s.server.Close(ctx); err != nil {
		return fmt.Errorf("failed to shut down pesit server: %w", err)
	}

	return nil
}

func (s *server) Connect(conn *pesit.ServerConnection) (pesit.TransferHandler, error) {
	if pass, err := s.getPassword(); err != nil {
		return nil, err
	} else if pass != "" {
		conn.SetServerPassword(pass)
	}

	if conn.HasCheckpoints() {
		if s.conf.DisableCheckpoints {
			conn.AllowCheckpoints(pesit.CheckpointDisabled, 0)
		} else {
			size := min(s.conf.CheckpointSize, conn.CheckpointSize())
			window := min(s.conf.CheckpointWindow, conn.CheckpointWindow())

			conn.AllowCheckpoints(size, window)
		}
	}

	if !s.conf.DisableRestart {
		conn.AllowRestart(true)
	}

	if conn.NewClientPassword() != "" {
		s.logger.Warningf("Connection from %q refused, clients are not allowed to change their password",
			conn.ClientLogin())

		return nil, pesit.NewDiagnostic(pesit.CodeMessageTypeRefused,
			"clients are not allowed to change their password")
	}

	user, authErr := s.authenticate(conn.ClientLogin(), conn.ClientPassword())
	if authErr != nil {
		return nil, authErr
	}

	s.logger.Debugf("Connection from %q successful", conn.ClientLogin())

	return &transferHandler{
		db:           s.db,
		logger:       s.logger,
		agent:        s.localAgent,
		account:      user,
		conf:         &s.conf.ServerConfig,
		tracer:       s.tracer,
		connFreetext: conn.FreeText(),
	}, nil
}

func (s *server) Release(conn *pesit.ServerConnection) {
	s.logger.Debugf("Connection closed to %v", conn)
}

var ErrPasswordDBError = errors.New("failed to retrieve the server password")

func (s *server) getPassword() (string, error) {
	var pass model.Credential
	if err := s.db.Get(&pass, "type=?", auth.Password).And(s.localAgent.GetCredCond()).Run(); err != nil {
		if database.IsNotFound(err) {
			return "", nil
		}

		s.logger.Errorf("Failed to retrieve the server password: %v", err)

		return "", ErrPasswordDBError
	}

	return pass.Value, nil
}

func (s *server) authenticate(login, password string) (*model.LocalAccount, error) {
	var user model.LocalAccount
	if err := s.db.Get(&user, "local_agent_id=? AND login=?", s.localAgent.ID,
		login).Run(); err != nil && !database.IsNotFound(err) {
		s.logger.Errorf("Failed to retrieve the local account: %v", err)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "failed to check the authentication")
	}

	res, authErr := user.Authenticate(s.db, s.localAgent, auth.Password, password)
	if authErr != nil {
		s.logger.Errorf("Failed to authenticate account %q: %v", login, authErr)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "failed to check the authentication")
	} else if !res.Success {
		s.logger.Warningf("authentication of account %q failed: %s", login, res.Reason)

		return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller, "invalid credentials")
	}

	return &user, nil
}
