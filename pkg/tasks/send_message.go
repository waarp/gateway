package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	SendMessageAckExpectedKey = "__ackExpected__"
	SendMessageAckSentKey     = "__ackSent__"
	SendMessageAckSentToKey   = "__ackSentTo__"
	SendMessageAckSentAsKey   = "__ackSentAs__"
	SendMessageAckSentOnKey   = "__ackSentOn__"
)

type MessageSender interface {
	SendMessage(db *database.DB, logger *log.Logger, client *model.Client,
		partner *model.RemoteAgent, account *model.RemoteAccount,
		transferInfo map[string]any, transferID, filename, message string) error
}

var (
	ErrSendMessageNoPartner   = errors.New(`missing "partner" argument`)
	ErrSendMessageNoAccount   = errors.New(`missing "account" argument`)
	ErrSendMessageNoClient    = errors.New(`missing "client" argument`)
	ErrSendMessageUnsupported = errors.New(`protocol does not support sending messages`)
)

// sendMessageTask is a post-task that sends a message to a remote
// partner. It is typically used for Store-and-Forward acknowledgments.
//
// Arguments:
//
//	partner   Remote partner name.
//	account   Remote account login.
//	account   Client name.
//	message   Message content. Supports variable substitution. Max 4096 chars.
type sendMessageTask struct {
	Partner string `json:"partner"`
	Account string `json:"account"`
	Client  string `json:"client"`
	Message string `json:"message"`

	partner model.RemoteAgent
	account model.RemoteAccount
	client  model.Client
}

func (t *sendMessageTask) ValidateDB(db database.ReadAccess, args map[string]string) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)
	}

	if t.Partner == "" {
		return ErrSendMessageNoPartner
	}

	if t.Account == "" {
		return ErrSendMessageNoAccount
	}

	if t.Client == "" {
		return ErrSendMessageNoClient
	}

	if err := db.Get(&t.client, "name=?", t.Client).Run(); err != nil {
		return fmt.Errorf("failed to retrieve client %q: %w", t.Client, err)
	}

	if err := db.Get(&t.partner, "name=?", t.Partner).Run(); err != nil {
		return fmt.Errorf("failed to retrieve partner %q: %w", t.Partner, err)
	}

	if account, err := t.partner.GetAccount(db, t.Account); err != nil {
		return fmt.Errorf("failed to retrieve account %q: %w", t.Account, err)
	} else {
		t.account = *account
	}

	return nil
}

//nolint:cyclop // the function has straightforward sequential logic
func (t *sendMessageTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, remote any,
) error {
	if err := t.ValidateDB(db, args); err != nil {
		return err
	}

	sender, isSender := remote.(MessageSender)
	if !isSender {
		return fmt.Errorf("%q: %w", t.partner.Protocol, ErrSendMessageUnsupported)
	}

	logger.Infof("SENDMESSAGE: sending message to partner %q as %q", t.Partner, t.Account)
	tID := transCtx.Transfer.RemoteTransferID

	filename := transCtx.Transfer.SrcFilename
	if filename == "" {
		filename = transCtx.Transfer.DestFilename
	}

	if err := sender.SendMessage(db, logger, &t.client, &t.partner, &t.account,
		transCtx.Transfer.TransferInfo, tID, filename, t.Message); err != nil {
		return fmt.Errorf("SENDMESSAGE failed: %w", err)
	}

	logger.Infof("SENDMESSAGE: F.MESSAGE sent successfully to %s as %s", t.Partner, t.Account)

	// Mark the transfer as ACK-sent for GUI visibility.
	if transCtx.Transfer.TransferInfo == nil {
		transCtx.Transfer.TransferInfo = map[string]any{}
	}

	delete(transCtx.Transfer.TransferInfo, SendMessageAckExpectedKey)
	transCtx.Transfer.TransferInfo[SendMessageAckSentKey] = true
	transCtx.Transfer.TransferInfo[SendMessageAckSentToKey] = t.Partner
	transCtx.Transfer.TransferInfo[SendMessageAckSentAsKey] = t.Account
	transCtx.Transfer.TransferInfo[SendMessageAckSentOnKey] = time.Now().Format(time.RFC3339)

	return nil
}
