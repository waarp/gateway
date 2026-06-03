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
	ErrSendMessageNotPeSIT = errors.New("SENDMESSAGE task is only supported with PeSIT protocol")
	ErrSendMessageNoTarget = errors.New("no target for SENDMESSAGE: set 'to' argument or configure replyTo on the partner")
)

type sendMessageTask struct {
	To      string `json:"to"`
	As      string `json:"as"`
	Message string `json:"message"`
}

func (t *sendMessageTask) parseArgs(args map[string]string) error {
	*t = sendMessageTask{}

	return utils.JSONConvert(args, t)
}

// Validate only checks JSON parsing — to/as are resolved at runtime from
// TransferInfo if not explicitly set.
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

	// Resolve target partner: explicit arg > __replyPartner__ from TransferInfo.
	partnerName := t.To
	if partnerName == "" {
		if rp, ok := transCtx.Transfer.TransferInfo["__replyPartner__"]; ok {
			partnerName = fmt.Sprintf("%v", rp)
		}
	}

	if partnerName == "" {
		return ErrSendMessageNoTarget
	}

	// Resolve account: explicit arg > __replyAccount__ > first account on partner.
	accountLogin := t.As
	if accountLogin == "" {
		if ra, ok := transCtx.Transfer.TransferInfo["__replyAccount__"]; ok {
			accountLogin = fmt.Sprintf("%v", ra)
		}
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", partnerName).Owner().Run(); err != nil {
		return fmt.Errorf("partner %q not found: %w", partnerName, err)
	}

	var account model.RemoteAccount
	if accountLogin != "" {
		if err := db.Get(&account, "remote_agent_id=? AND login=?",
			partner.ID, accountLogin).Run(); err != nil {
			logger.Warningf("Account %q not found on partner %q, trying first available",
				accountLogin, partnerName)
		}
	}

	if account.ID == 0 {
		if err := db.Get(&account, "remote_agent_id=?", partner.ID).Run(); err != nil {
			return fmt.Errorf("no account found on partner %q: %w", partnerName, err)
		}
	}

	// Build message content.
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

	// Retrieve account password.
	password := getRemoteAccountPassword(db, &account)

	// Connect to partner and send F.MESSAGE.
	if err := doSendPeSITMessage(&partner, &account, password,
		transferID, message); err != nil {
		logger.Errorf("SENDMESSAGE to %q failed: %v", partnerName, err)

		return err
	}

	logger.Infof("F.MESSAGE sent to %q as %q (transferID=%d, message=%q)",
		partnerName, account.Login, transferID, message)

	return nil
}

func getRemoteAccountPassword(db database.ReadAccess, account *model.RemoteAccount) string {
	var cred model.Credential
	if err := db.Get(&cred, "remote_account_id=? AND type=?",
		account.ID, "password").Run(); err == nil {
		return cred.Value
	}

	return ""
}

func doSendPeSITMessage(partner *model.RemoteAgent, account *model.RemoteAccount,
	password string, transferID uint32, message string,
) error {
	client := pesit.NewClient(account.Login, password, partner.Name)
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
