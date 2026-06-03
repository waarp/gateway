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
//
// When relayMessages is enabled in the server config, the handler also
// attempts to relay the message upstream through the Store & Forward chain.
func (s *server) HandleMessage(conn *pesit.ServerConnection, msg pesit.MessageRequest) error {
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

	followStr := fmt.Sprintf("%v", followRaw)

	var inTrans model.Transfer
	if err := s.db.Get(&inTrans, "remote_transfer_id=?", followStr).
		Owner().OrderBy("start", false).Run(); err != nil {
		// Try by ID
		if err2 := s.db.Get(&inTrans, "id=?", followStr).
			Owner().Run(); err2 != nil {
			s.logger.Debugf("Cannot find upstream transfer for __followID__=%s", followStr)

			return
		}
	}

	// The incoming transfer (A→B) is a server transfer with a local_account_id.
	// The requester is the client that connected to us. To relay, we need to
	// find a partner definition that we can connect TO.
	//
	// Resolution strategy (in order of priority):
	//
	// The standard PeSIT Store & Forward mechanism uses PI 61 (CustomerID)
	// and PI 62 (BankID) to identify the originator through the relay chain.
	// This is the standard approach used by PeSIT products on the market.
	//
	// 1. customerID (PI 61) from the F.MESSAGE — standard PeSIT S&F
	// 2. bankID (PI 62) from the F.MESSAGE — standard PeSIT S&F
	// 3. customerID from the original incoming transfer — preserved via TransferInfo
	// 4. __replyPartner__ from PI 99 freetext ("REPLY=partner:account") — Waarp override
	// 5. Login of the LocalAccount (convention: partner name = client login)
	candidates := []string{}

	// Priority 1-2: PI 61/62 from the F.MESSAGE (standard PeSIT mechanism)
	candidates = append(candidates, msg.CustomerID, msg.BankID)

	// Priority 3: customerID from incoming transfer
	if cid, ok := inTrans.TransferInfo[customerIDKey]; ok {
		candidates = append(candidates, fmt.Sprintf("%v", cid))
	}

	// Priority 4: __replyPartner__ from PI 99 convention (Waarp-specific override)
	if rp, ok := inTrans.TransferInfo[replyPartnerKey]; ok {
		candidates = append(candidates, fmt.Sprintf("%v", rp))
	}

	// Priority 5: LocalAccount login (naming convention fallback)
	if inTrans.LocalAccountID.Valid {
		var acct model.LocalAccount
		if err := s.db.Get(&acct, "id=?", inTrans.LocalAccountID.Int64).Run(); err == nil {
			candidates = append(candidates, acct.Login)
		}
	}

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

		return
	}

	// Resolve account: __replyAccount__ > first account on partner
	var accountLogin string
	if ra, ok := inTrans.TransferInfo[replyAccountKey]; ok {
		accountLogin = fmt.Sprintf("%v", ra)
	}

	if partner.Protocol != Pesit && partner.Protocol != PesitTLS {
		s.logger.Debugf("Partner %q is not PeSIT, skipping F.MESSAGE relay", partner.Name)

		return
	}

	// Find an account on the partner: use __replyAccount__ if set, else first available.
	var account model.RemoteAccount

	if accountLogin != "" {
		if err := s.db.Get(&account, "remote_agent_id=? AND login=?",
			partner.ID, accountLogin).Run(); err != nil {
			s.logger.Debugf("Account %q not found on partner %q, trying first available",
				accountLogin, partner.Name)
		}
	}

	if account.ID == 0 {
		if err := s.db.Get(&account, "remote_agent_id=?", partner.ID).Run(); err != nil {
			s.logger.Debugf("No account found on partner %q for F.MESSAGE relay", partner.Name)

			return
		}
	}

	// Get account password.
	var cred model.Credential
	password := ""

	if err := s.db.Get(&cred, "remote_account_id=? AND type=?",
		account.ID, "password").Run(); err == nil {
		password = cred.Value
	}

	// Build the upstream transfer ID from the incoming transfer.
	var upstreamTransferID uint32
	if rid := inTrans.RemoteTransferID; rid != "" {
		if id, err := strconv.ParseUint(rid, 10, 32); err == nil {
			upstreamTransferID = uint32(id)
		}
	}

	// Send F.MESSAGE upstream.
	s.logger.Infof("Relaying F.MESSAGE to partner %q (transferID=%d)",
		partner.Name, upstreamTransferID)

	go func() {
		if err := sendRelayMessage(&partner, &account, password,
			upstreamTransferID, msg.Message); err != nil {
			s.logger.Warningf("Failed to relay F.MESSAGE to %q: %v", partner.Name, err)
		} else {
			s.logger.Infof("F.MESSAGE relayed to %q successfully", partner.Name)
		}
	}()
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
