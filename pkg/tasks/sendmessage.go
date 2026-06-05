package tasks

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //same pattern as NewClientPipeline - breaks import cycle

var SendPeSITMessage func(db *database.DB, partnerName, accountLogin string,
	transferID uint32, message string, logger *log.Logger,
	customerID, bankID string) error

var (
	ErrSendMessageNoPartner = errors.New(`no target for SENDMESSAGE: set "partner" argument or configure replyTo on the partner`)

	ErrSendMessagePartnerNotFound = errors.New("sendmessage partner not found")

	ErrSendMessageAccountNotFound = errors.New("sendmessage account not found")

	ErrSendMessageNotPeSIT = errors.New("sendmessage only works with PeSIT partners")

	ErrSendMessageNotWired = errors.New("SENDMESSAGE not available: PeSIT module not loaded")
)

// sendMessageTask is a post-task that sends a PeSIT F.MESSAGE to a remote
// partner. Typically used for application-level acknowledgments (ACK).
//
// Partner and account are resolved automatically from the "Reply To (ACK)"
// address configured on the sending partner if not explicitly provided.
//
// Arguments:
//
//	partner     (optional) Remote partner name. Auto-resolved from Reply To if empty.
//	account     (optional) Remote account login. Defaults to first account of the partner.
//	message     (optional) Message content. Supports variable substitution. Max 4096 chars.
//	transferId  (optional) PeSIT transfer ID to reference in the F.MESSAGE.
//	passthrough (optional) If "true", skip sending (S&F relay: only the final destination sends the ACK).
//	customerID  (optional) Applicative identifier. Propagates origin identity through S&F relay chains.
//	bankID      (optional) Secondary applicative identifier. Fallback for origin identity in S&F.
type sendMessageTask struct {
	Partner     string `json:"partner"`
	Account     string `json:"account"`
	Message     string `json:"message"`
	TransferID  string `json:"transferId"`
	Passthrough string `json:"passthrough"`
	CustomerID  string `json:"customerID"`
	BankID      string `json:"bankID"`
}

// Validate checks JSON parsing. Partner/account are resolved at runtime
// from the replyTo configuration if not explicitly set.
func (t *sendMessageTask) Validate(args map[string]string) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)
	}

	return nil
}

func (t *sendMessageTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)
	}

	// Passthrough mode: in S&F, the relay node (B) must NOT send an ACK.
	// Only the final destination (C) should emit the ACK.
	if t.Passthrough == "true" {
		logger.Debug("SENDMESSAGE skipped: passthrough mode (S&F relay)")
		return nil
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

	// If no explicit transferID, use the current transfer's remote ID.
	if tid == 0 {
		tid = parseMessageTransferID(transCtx.Transfer.RemoteTransferID, logger)
	}

	// Default message if none configured.
	message := t.Message
	if message == "" {
		filename := path.Base(transCtx.Transfer.LocalPath)
		message = fmt.Sprintf("ACK %s %s", transCtx.Rule.Name, filename)
	}

	logger.Infof("SENDMESSAGE: sending F.MESSAGE to partner %q as %q", partner.Name, account.Login)

	if err := SendPeSITMessage(db, partner.Name, account.Login, tid, message, logger,
		t.CustomerID, t.BankID); err != nil {
		return fmt.Errorf("SENDMESSAGE failed: %w", err)
	}

	logger.Infof("SENDMESSAGE: F.MESSAGE sent successfully to %s/%s", partner.Name, account.Login)

	// Record the ACK as SENT in ack_tracking.
	ack := &model.AckTracking{
		TransferID: transCtx.Transfer.ID,
		RemoteID:   transCtx.Transfer.RemoteTransferID,
		IsSend:     false,
		State:      model.AckStateSent,
		Partner:    partner.Name,
		Account:    account.Login,
		Message:    message,
	}

	if err := model.InsertAckTracking(db, ack); err != nil {
		logger.Warningf("Failed to insert ack_tracking(SENT): %v", err)
	}

	return nil
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
