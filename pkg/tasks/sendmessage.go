package tasks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //same pattern as NewClientPipeline - breaks import cycle

var SendPeSITMessage func(db *database.DB, partnerName, accountLogin string,

	transferID uint32, message string, logger *log.Logger) error

var (
	ErrSendMessageNoPartner = errors.New(`no target for SENDMESSAGE: set "partner" argument or configure replyTo on the partner`)

	ErrSendMessagePartnerNotFound = errors.New("sendmessage partner not found")

	ErrSendMessageAccountNotFound = errors.New("sendmessage account not found")

	ErrSendMessageNotPeSIT = errors.New("sendmessage only works with PeSIT partners")

	ErrSendMessageNotWired = errors.New("SENDMESSAGE not available: PeSIT module not loaded")
)

// sendMessageTask is a post-task that sends a F.MESSAGE (PeSIT) to a remote

// partner. It is typically used for Store-and-Forward acknowledgments.

//

// The task is conditional: if the "condition" argument is set, the task checks

// that the corresponding TransferInfo key exists and equals "1". If not, the

// task silently succeeds without sending anything.

//

// Arguments:

//

//	partner   (optional) Remote partner name. Resolved from __replyPartner__ if empty. Supports variable substitution.

//	account   (optional) Remote account login. Defaults to first account of the partner.

//	message   (optional) Message content. Supports variable substitution. Max 4096 chars.

//	transferId (optional) Transfer ID to reference in the F.MESSAGE. Supports variables.

//	condition (optional) TransferInfo key to check. If set and value != "1", skip.

type sendMessageTask struct {
	Partner string `json:"partner"`

	Account string `json:"account"`

	Message string `json:"message"`

	TransferID string `json:"transferId"`

	Condition string `json:"condition"`
}

// Validate only checks JSON parsing -- partner/account are resolved at runtime

// from TransferInfo if not explicitly set.

func (t *sendMessageTask) Validate(args map[string]string) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)
	}

	// partner is no longer required at validation time: it can be resolved

	// at runtime from __replyPartner__ in TransferInfo.

	return nil
}

func (t *sendMessageTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)
	}

	if skip, err := t.checkCondition(logger, transCtx); skip || err != nil {
		return err
	}

	partnerName, accountLogin := t.resolveTarget(transCtx)
	if partnerName == "" {
		logger.Debug("SENDMESSAGE skipped: no target partner")
		return &WarningError{msg: "no ACK requested (no reply partner configured)"}
	}

	partner, account, err := t.resolvePartnerAccount(db, logger, partnerName, accountLogin)
	if err != nil {
		return err
	}

	if SendPeSITMessage == nil {
		return ErrSendMessageNotWired
	}

	tid := parseMessageTransferID(t.TransferID, logger)

	logger.Infof("SENDMESSAGE: sending F.MESSAGE to partner %q as %q", partner.Name, account.Login)

	if err := SendPeSITMessage(db, partner.Name, account.Login, tid, t.Message, logger); err != nil {
		return fmt.Errorf("SENDMESSAGE failed: %w", err)
	}

	logger.Infof("SENDMESSAGE: F.MESSAGE sent successfully to %s/%s", partner.Name, account.Login)

	// Mark the transfer as ACK-sent for GUI visibility.
	transCtx.Transfer.TransferInfo["__ackSent__"] = "true"
	transCtx.Transfer.TransferInfo["__ackSentTo__"] = partner.Name
	transCtx.Transfer.TransferInfo["__ackSentAs__"] = account.Login
	transCtx.Transfer.TransferInfo["__ackSentAt__"] = time.Now().UTC().Format(time.RFC3339)

	return nil
}

func (t *sendMessageTask) checkCondition(logger *log.Logger,
	transCtx *model.TransferContext,
) (skip bool, err error) {
	if t.Condition == "" {
		return false, nil
	}

	val, ok := transCtx.Transfer.TransferInfo[t.Condition]
	if !ok || fmt.Sprint(val) != "1" {
		logger.Debugf("SENDMESSAGE skipped: condition %q not met", t.Condition)
		return true, nil
	}

	return false, nil
}

func (t *sendMessageTask) resolveTarget(transCtx *model.TransferContext) (string, string) {
	partnerName := t.Partner
	if partnerName == "" {
		if rp, ok := transCtx.Transfer.TransferInfo["__replyPartner__"]; ok {
			partnerName = fmt.Sprintf("%v", rp)
		}
	}

	accountLogin := t.Account
	if accountLogin == "" {
		if ra, ok := transCtx.Transfer.TransferInfo["__replyAccount__"]; ok {
			accountLogin = fmt.Sprintf("%v", ra)
		}
	}

	return partnerName, accountLogin
}

func (t *sendMessageTask) resolvePartnerAccount(db *database.DB, logger *log.Logger,
	partnerName, accountLogin string,
) (*model.RemoteAgent, *model.RemoteAccount, error) {
	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", partnerName).Owner().
		Run(); database.IsNotFound(err) {
		return nil, nil, fmt.Errorf("%w: %q", ErrSendMessagePartnerNotFound, partnerName)
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve partner %q: %w", partnerName, err)
	}

	if partner.Protocol != "pesit" && partner.Protocol != "pesit-tls" {
		return nil, nil, fmt.Errorf("%w: partner %q uses protocol %q",
			ErrSendMessageNotPeSIT, partnerName, partner.Protocol)
	}

	account, err := resolveMessageAccount(db, logger, &partner, accountLogin)
	if err != nil {
		return nil, nil, err
	}

	return &partner, account, nil
}

func resolveMessageAccount(db *database.DB, logger *log.Logger,
	partner *model.RemoteAgent, accountLogin string,
) (*model.RemoteAccount, error) {
	var account model.RemoteAccount
	if accountLogin != "" {
		if err := db.Get(&account, "login=?", accountLogin).
			And("remote_agent_id=?", partner.ID).Run(); err == nil {
			return &account, nil
		}
		logger.Warningf("Account %q not found on partner %q, trying first available",
			accountLogin, partner.Name)
	}

	if err := db.Get(&account, "remote_agent_id=?", partner.ID).
		Run(); database.IsNotFound(err) {
		return nil, fmt.Errorf("%w: no account found for partner %q",
			ErrSendMessageAccountNotFound, partner.Name)
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve account for partner %q: %w", partner.Name, err)
	}

	return &account, nil
}

func parseMessageTransferID(tidStr string, logger *log.Logger) uint32 {
	if tidStr == "" {
		return 0
	}
	tid64, err := strconv.ParseUint(tidStr, 10, 32)
	if err != nil {
		logger.Warningf("SENDMESSAGE: invalid transferId %q, using 0", tidStr)
		return 0
	}
	return uint32(tid64)
}
