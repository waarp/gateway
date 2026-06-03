package pesit

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
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
	conf       *ServerConfigTLS
}

func (s *server) listen() (string, error) {
	s.server = pesit.NewServer(s)
	s.server.Logger = s.logger.AsStdLogger(log.LevelDebug)

	if s.conf.DisablePreConnection {
		s.server.SetPreConnectionUsage(false)
	}
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

// HandleMessage implements pesit.MessageHandler to accept incoming F.MESSAGE.
// The message metadata is logged and stored in the database as a transfer
// history entry for traceability. This prevents the ABORT that would occur
// if the interface were not implemented.
func (s *server) HandleMessage(conn *pesit.ServerConnection, msg pesit.MessageRequest) error {
	s.logger.Infof("F.MESSAGE received from %q: transferID=%d customerID=%q bankID=%q message=%q",
		msg.ClientLogin, msg.TransferID, msg.CustomerID, msg.BankID, msg.Message)

	// Store the message as a transfer info entry for traceability.
	// Find the original transfer by remote transfer ID if possible.
	if msg.TransferID != 0 {
		remoteID := strconv.FormatUint(uint64(msg.TransferID), 10)

		var trans model.Transfer
		if err := s.db.Get(&trans, "remote_transfer_id=?", remoteID).
			Owner().OrderBy("start", false).Run(); err == nil {
			// Update the transfer's info with the ACK message.
			if trans.TransferInfo == nil {
				trans.TransferInfo = make(map[string]any)
			}

			trans.TransferInfo["__messageACK__"] = msg.Message
			trans.TransferInfo["__messageCustomerID__"] = msg.CustomerID
			trans.TransferInfo["__messageBankID__"] = msg.BankID

			if err := s.db.Update(&trans).Cols("transfer_info").Run(); err != nil {
				s.logger.Warningf("Failed to store F.MESSAGE info on transfer %d: %v",
					trans.ID, err)
			} else {
				s.logger.Infof("F.MESSAGE ACK stored on transfer %d", trans.ID)
			}
		} else {
			s.logger.Debugf("No matching transfer found for F.MESSAGE transferID=%d", msg.TransferID)
		}
	}

	return nil // Accept the message (ACK with success)
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
