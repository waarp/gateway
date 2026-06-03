package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"strconv"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	protoPeSIT    = "pesit"
	protoPeSITTLS = "pesit-tls"
)

var (
	ErrSendMessageNoPartner = errors.New(`missing "to" argument (partner name)`)
	ErrSendMessageNoAccount = errors.New(`missing "as" argument (account login)`)
	ErrSendMessageNotPeSIT  = errors.New("SENDMESSAGE task is only supported with PeSIT protocol")
)

type sendMessageTask struct {
	To      string `json:"to"`
	As      string `json:"as"`
	Message string `json:"message"`
}

func (t *sendMessageTask) parseArgs(args map[string]string) error {
	*t = sendMessageTask{}
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse SENDMESSAGE arguments: %w", err)
	}

	if t.To == "" {
		return ErrSendMessageNoPartner
	}

	if t.As == "" {
		return ErrSendMessageNoAccount
	}

	return nil
}

func (t *sendMessageTask) Validate(args map[string]string) error {
	return t.parseArgs(args)
}

func (t *sendMessageTask) Run(ctx context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.parseArgs(args); err != nil {
		return err
	}

	// Check protocol: SENDMESSAGE is PeSIT-only.
	protocol := getTransferProtocol(transCtx)
	if protocol != protoPeSIT && protocol != protoPeSITTLS {
		logger.Warningf("SENDMESSAGE skipped: transfer uses protocol %q (PeSIT required)", protocol)

		return &WarningError{msg: ErrSendMessageNotPeSIT.Error()}
	}

	// Resolve partner and account.
	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", t.To).Owner().Run(); err != nil {
		return fmt.Errorf("partner %q not found: %w", t.To, err)
	}

	var account model.RemoteAccount
	if err := db.Get(&account, "login=? AND remote_agent_id=?", t.As, partner.ID).Run(); err != nil {
		return fmt.Errorf("account %q on partner %q not found: %w", t.As, t.To, err)
	}

	// Build message content (use provided message or default ACK).
	message := t.Message
	if message == "" {
		message = fmt.Sprintf("ACK transfer %d", transCtx.Transfer.ID)
	}

	// Get transfer ID to reference in the message.
	var transferID uint32

	remoteID := transCtx.Transfer.RemoteTransferID
	if remoteID != "" {
		if id, err := strconv.ParseUint(remoteID, 10, 32); err == nil {
			transferID = uint32(id)
		}
	}

	if transferID == 0 {
		transferID = uint32(transCtx.Transfer.ID)
	}

	// Retrieve account password from credentials.
	password, passErr := getAccountPassword(db, &account)
	if passErr != nil {
		logger.Warningf("No password credential found for account %q: %v", t.As, passErr)
	}

	// Connect to partner and send F.MESSAGE.
	if err := t.sendPeSITMessage(logger, &partner, &account, password, transferID, message); err != nil {
		logger.Errorf("SENDMESSAGE failed: %v", err)

		return err
	}

	logger.Infof("F.MESSAGE sent to %q as %q (transferID=%d)", t.To, t.As, transferID)

	return nil
}

func getAccountPassword(db database.ReadAccess, account *model.RemoteAccount) (string, error) {
	var creds model.Credentials
	if err := db.Select(&creds).Where("remote_account_id=? AND type=?",
		account.ID, "password").Run(); err != nil {
		return "", fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	for _, cred := range creds {
		if cred.Type == "password" {
			return cred.Value, nil
		}
	}

	return "", nil
}

func (t *sendMessageTask) sendPeSITMessage(logger *log.Logger,
	partner *model.RemoteAgent, account *model.RemoteAccount, password string,
	transferID uint32, message string,
) error {
	serverLogin := partner.Name

	client := pesit.NewClient(account.Login, password, serverLogin)
	client.SetPreConnectionUsage(false)
	client.Logger = stdlog.New(io.Discard, "", 0)

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

// getTransferProtocol returns the protocol of the current transfer.
func getTransferProtocol(transCtx *model.TransferContext) string {
	if transCtx.RemoteAgent != nil && transCtx.RemoteAgent.Protocol != "" {
		return transCtx.RemoteAgent.Protocol
	}

	if transCtx.LocalAgent != nil && transCtx.LocalAgent.Protocol != "" {
		return transCtx.LocalAgent.Protocol
	}

	return ""
}
