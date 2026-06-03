package pesit

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/lib/pesit"
)

type server struct {
	db *database.DB

	logger *log.Logger

	tracer func() pipeline.Trace

	server *pesit.Server

	state utils.State

	localAgent *model.LocalAgent

	conf *ServerConfigTLS
}

func (s *server) listen() (string, error) {
	s.server = pesit.NewServer(s)

	s.server.Logger = s.logger.AsStdLogger(log.LevelDebug)

	s.server.NetworkTrace = s.logger.AsStdLogger(log.LevelTrace)

	// Disable pre-connection in PeSIT-TLS mode or when explicitly configured.

	// Most TLS partners do not send the 24-byte EBCDIC pre-connection message.

	if s.localAgent.Protocol == PesitTLS || s.conf.DisablePreConnection {
		s.server.SetPreConnectionUsage(false)
	}

	realAddr := conf.GetRealAddress(s.localAgent.Address.Host,

		utils.FormatUint(s.localAgent.Address.Port))

	var (
		list net.Listener

		listErr error
	)

	if s.localAgent.Protocol == PesitTLS {

		cipherIDs, _ := resolveCipherSuites(s.conf.CipherSuites)

		tlsConfig := &tls.Config{
			MinVersion:            s.conf.MinTLSVersion.TLS(),
			ClientAuth:            tlsRequireClientCert(s.conf.TLSClientAuth),
			GetCertificate:        s.getCertificate,
			VerifyPeerCertificate: auth.VerifyClientCert(s.db, s.logger, s.localAgent),
			CipherSuites:          cipherIDs, // nil = Go defaults
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

	if s.conf.ProtocolTimeout > 0 {
		conn.SetMonitoringTimeout(s.conf.ProtocolTimeout)
	}

	if conn.NewClientPassword() != "" {

		s.logger.Warningf("Connection from %q refused, clients are not allowed to change their password",

			conn.ClientLogin())

		return nil, pesit.NewDiagnostic(pesit.CodeMessageTypeRefused,

			"clients are not allowed to change their password")

	}

	// Authenticate the client. In TLS with tlsClientAuth="required" or
	// "optional", the certificate has already been validated by
	// VerifyPeerCertificate (which matches CN to a local account). We trust
	// the PeSIT login (PI 3) as the account identifier without checking the
	// password, since the TLS certificate is the proof of identity.
	var user *model.LocalAccount

	certAuthMode := s.conf.TLSClientAuth
	isTLS := s.localAgent.Protocol == PesitTLS
	emptyPassword := conn.ClientPassword() == ""

	switch {
	case certAuthMode == TLSClientAuthRequired && isTLS:
		// Certificate required: trust the TLS cert, ignore PeSIT password.
		acc, err := s.findAccountByLogin(conn.ClientLogin())
		if err != nil {
			return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller,
				fmt.Sprintf("certificate auth: %v", err))
		}
		s.logger.Debugf("TLS cert auth (required): accepted %q", conn.ClientLogin())
		user = acc

	case certAuthMode == TLSClientAuthOptional && isTLS && emptyPassword:
		// Certificate optional + no password: trust the TLS cert.
		acc, err := s.findAccountByLogin(conn.ClientLogin())
		if err != nil {
			return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller,
				fmt.Sprintf("certificate auth: %v", err))
		}
		s.logger.Debugf("TLS cert auth (optional): accepted %q (no password)", conn.ClientLogin())
		user = acc

	case isTLS && emptyPassword:
		// Legacy: TLS mutual auth with empty password — accept based on login.
		acc, err := s.findAccountByLogin(conn.ClientLogin())
		if err != nil {
			s.logger.Warningf("TLS mutual auth: unknown login %q", conn.ClientLogin())
			return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller, "unknown login")
		}
		s.logger.Debugf("TLS mutual auth accepted for %q (empty password)", conn.ClientLogin())
		user = acc

	default:
		// Standard authentication: login + password.
		var authErr error
		user, authErr = s.authenticate(conn.ClientLogin(), conn.ClientPassword())
		if authErr != nil {
			return nil, authErr
		}
	}

	s.logger.Debugf("Connection from %q successful", conn.ClientLogin())

	return &transferHandler{
		db: s.db,

		logger: s.logger,

		agent: s.localAgent,

		account: user,

		conf: &s.conf.ServerConfig,

		tracer: s.tracer,

		connFreetext: conn.FreeText(),
	}, nil
}

func (s *server) Release(conn *pesit.ServerConnection) {
	s.logger.Debugf("Connection closed to %v", conn)
}

func (s *server) findAccountByLogin(login string) (*model.LocalAccount, error) {
	var acc model.LocalAccount
	if err := s.db.Get(&acc, "local_agent_id=? AND login=?",
		s.localAgent.ID, login).Run(); err != nil {
		return nil, fmt.Errorf("account %q not found: %w", login, err)
	}

	return &acc, nil
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

// HandleMessage implements pesit.MessageHandler to accept incoming F.MESSAGE.

// The message metadata is logged and stored in the database as a transfer

// history entry for traceability. This prevents the ABORT that would occur

// if the interface were not implemented.

//

// When relayMessages is enabled in the server config, the handler also

// attempts to relay the message upstream through the Store & Forward chain.

func (s *server) HandleMessage(_ *pesit.ServerConnection, msg pesit.MessageRequest) error {
	s.logger.Infof("F.MESSAGE received from %q: transferID=%d customerID=%q bankID=%q message=%q",

		msg.ClientLogin, msg.TransferID, msg.CustomerID, msg.BankID, msg.Message)

	if msg.TransferID == 0 {
		return nil
	}

	remoteID := strconv.FormatUint(uint64(msg.TransferID), 10)

	// Find the outgoing transfer that this ACK references.

	var outTrans model.Transfer

	if err := s.db.Get(&outTrans, "remote_transfer_id=?", remoteID).
		Owner().OrderBy("start", false).Run(); err != nil {

		s.logger.Debugf("No matching transfer found for F.MESSAGE transferID=%d", msg.TransferID)

		return nil

	}

	// Store the ACK info on the outgoing transfer.

	if outTrans.TransferInfo == nil {
		outTrans.TransferInfo = make(map[string]any)
	}

	outTrans.TransferInfo["__messageACK__"] = msg.Message

	outTrans.TransferInfo["__messageCustomerID__"] = msg.CustomerID

	outTrans.TransferInfo["__messageBankID__"] = msg.BankID

	if err := s.db.Update(&outTrans).Cols("transfer_info").Run(); err != nil {
		s.logger.Warningf("Failed to store F.MESSAGE info on transfer %d: %v",

			outTrans.ID, err)
	} else {
		s.logger.Infof("F.MESSAGE ACK stored on transfer %d", outTrans.ID)
	}

	// If relay is enabled, try to forward the message upstream.

	if s.conf.RelayMessages {
		s.relayMessage(&outTrans, msg)
	}

	return nil
}

// relayMessage attempts to relay a F.MESSAGE upstream through the Store &

// Forward chain. It follows the __followID__ link to find the original

// incoming transfer, resolves the upstream partner, and sends the message.

func (s *server) relayMessage(outTrans *model.Transfer, msg pesit.MessageRequest) {
	// Follow the chain: outgoing transfer (B→C) → __followID__ → incoming transfer (A→B)
	followRaw, ok := outTrans.TransferInfo[model.FollowID]
	if !ok {
		s.logger.Debug("No __followID__ on transfer, cannot relay F.MESSAGE upstream")
		return
	}

	inTrans, found := s.findUpstreamTransfer(fmt.Sprintf("%v", followRaw))
	if !found {
		return
	}

	partner, account, password, ok := s.resolveUpstreamTarget(inTrans, msg)
	if !ok {
		return
	}

	upstreamTransferID := parseTransferID(inTrans.RemoteTransferID)

	s.logger.Infof("Relaying F.MESSAGE to partner %q (transferID=%d)",
		partner.Name, upstreamTransferID)

	go func() {
		if err := sendRelayMessage(partner, account, password,
			upstreamTransferID, msg.Message); err != nil {
			s.logger.Warningf("Failed to relay F.MESSAGE to %q: %v", partner.Name, err)
		} else {
			s.logger.Infof("F.MESSAGE relayed to %q successfully", partner.Name)
		}
	}()
}

func (s *server) findUpstreamTransfer(followStr string) (*model.Transfer, bool) {
	var inTrans model.Transfer
	if err := s.db.Get(&inTrans, "remote_transfer_id=?", followStr).
		Owner().OrderBy("start", false).Run(); err == nil {
		return &inTrans, true
	}

	if err := s.db.Get(&inTrans, "id=?", followStr).Owner().Run(); err == nil {
		return &inTrans, true
	}

	s.logger.Debugf("Cannot find upstream transfer for __followID__=%s", followStr)
	return nil, false
}

//nolint:cyclop // resolution logic requires checking multiple sources in order
func (s *server) resolveUpstreamTarget(inTrans *model.Transfer, msg pesit.MessageRequest,
) (*model.RemoteAgent, *model.RemoteAccount, string, bool) {
	// Build candidate list: PI 61/62, customerID, replyPartner, login convention
	candidates := []string{msg.CustomerID, msg.BankID}

	if cid, ok := inTrans.TransferInfo[customerIDKey]; ok {
		candidates = append(candidates, fmt.Sprintf("%v", cid))
	}
	if rp, ok := inTrans.TransferInfo[replyPartnerKey]; ok {
		candidates = append(candidates, fmt.Sprintf("%v", rp))
	}
	if inTrans.LocalAccountID.Valid {
		var acct model.LocalAccount
		if err := s.db.Get(&acct, "id=?", inTrans.LocalAccountID.Int64).Run(); err == nil {
			candidates = append(candidates, acct.Login)
		}
	}

	// Find the first matching PeSIT partner
	var partner model.RemoteAgent
	found := false
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if err := s.db.Get(&partner, "name=?", name).Owner().Run(); err == nil {
			found = true
			break
		}
	}
	if !found {
		s.logger.Debugf("No upstream partner found for F.MESSAGE relay (tried: %v)", candidates)
		return nil, nil, "", false
	}
	if partner.Protocol != Pesit && partner.Protocol != PesitTLS {
		s.logger.Debugf("Partner %q is not PeSIT, skipping F.MESSAGE relay", partner.Name)
		return nil, nil, "", false
	}

	// Resolve account
	account, password := s.resolveUpstreamAccount(&partner, inTrans)
	if account == nil {
		return nil, nil, "", false
	}
	return &partner, account, password, true
}

func (s *server) resolveUpstreamAccount(partner *model.RemoteAgent, inTrans *model.Transfer,
) (*model.RemoteAccount, string) {
	var account model.RemoteAccount
	if ra, ok := inTrans.TransferInfo[replyAccountKey]; ok {
		login := fmt.Sprintf("%v", ra)
		if err := s.db.Get(&account, "remote_agent_id=? AND login=?",
			partner.ID, login).Run(); err == nil {
			return &account, s.getAccountPassword(&account)
		}
	}
	if err := s.db.Get(&account, "remote_agent_id=?", partner.ID).Run(); err != nil {
		s.logger.Debugf("No account found on partner %q for F.MESSAGE relay", partner.Name)
		return nil, ""
	}
	return &account, s.getAccountPassword(&account)
}

func (s *server) getAccountPassword(account *model.RemoteAccount) string {
	var cred model.Credential
	if err := s.db.Get(&cred, "remote_account_id=? AND type=?",
		account.ID, "password").Run(); err == nil {
		return cred.Value
	}
	return ""
}

func parseTransferID(rid string) uint32 {
	if rid == "" {
		return 0
	}
	if id, err := strconv.ParseUint(rid, 10, 32); err == nil {
		return uint32(id)
	}
	return 0
}

func sendRelayMessage(partner *model.RemoteAgent, account *model.RemoteAccount,

	password string, transferID uint32, message string,
) error {
	client := pesit.NewClient(account.Login, password, partner.Name)

	client.SetPreConnectionUsage(false)

	addr := fmt.Sprintf("%s:%d", partner.Address.Host, partner.Address.Port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	if err := client.Connect(conn); err != nil {

		conn.Close()

		return fmt.Errorf("failed to establish PeSIT connection: %w", err)

	}

	defer client.Close(nil)

	if err := client.SendMessage(transferID, message); err != nil {
		return fmt.Errorf("failed to send F.MESSAGE: %w", err)
	}

	return nil
}
