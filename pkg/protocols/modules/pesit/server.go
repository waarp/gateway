package pesit

import (
	"context"
	"errors"
	"fmt"
	"net"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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
	conf       ServerConfigTLS
}

func (s *server) listen() (string, error) {
	s.server = pesit.NewServer(s)
	s.server.Logger = s.logger.AsStdLogger(log.LevelDebug)
	s.server.NetworkTrace = s.logger.AsStdLogger(log.LevelTrace)
	realAddr := s.db.Config.Overrides.GetRealAddress(s.localAgent.Address.Host,
		utils.FormatUint(s.localAgent.Address.Port))

	var (
		list    net.Listener
		listErr error
	)

	if s.localAgent.Protocol == PesitTLS {
		tlsConfig := protoutils.GetServerTLSConfig(s.db, s.logger, s.localAgent.ID)
		list, listErr = protoutils.ListenTLS("tcp", realAddr, tlsConfig)
	} else {
		list, listErr = protoutils.Listen("tcp", realAddr)
	}

	if listErr != nil {
		return "", fmt.Errorf("failed to open listener: %w", listErr)
	}

	go func() {
		if err := s.server.Serve(list); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.logger.Errorf("unexpected error: %v", err)
				s.state.Set(utils.StateError, err.Error())
			}
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

	user, usErr := s.authenticate(conn)
	if usErr != nil {
		return nil, usErr
	}

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

func (s *server) authenticate(conn *pesit.ServerConnection) (*model.LocalAccount, error) {
	authenticated := false
	login := conn.ClientLogin()
	password := conn.ClientPassword()

	user, usErr := s.localAgent.GetAccount(s.db, conn.ClientLogin())
	if usErr != nil && !database.IsNotFound(usErr) {
		s.logger.Errorf("Failed to retrieve account from database: %v", usErr)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "database error")
	}

	if tlsState, isTLS := conn.TLSConnectionState(); isTLS {
		if len(tlsState.PeerCertificates) > 0 {
			if protoutils.CheckClientCert(user, tlsState.PeerCertificates) {
				authenticated = true
			}
		}
	}

	if pwdRes, pwdErr := user.Authenticate(s.db, auth.Password, password); pwdErr != nil {
		s.logger.Errorf("Failed to authenticate account %q: %v", login, pwdErr)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "failed to check the authentication")
	} else if pwdRes.Success {
		authenticated = true
	}

	if !authenticated {
		s.logger.Warningf("authentication of account %q failed", login)

		return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller, "invalid credentials")
	}

	s.logger.Debugf("Connection from %q successful", conn.ClientLogin())

	return user, nil
}
