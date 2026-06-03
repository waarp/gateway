package tasks

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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

//nolint:cyclop // the function has straightforward sequential logic

func (t *sendMessageTask) Run(_ context.Context, args map[string]string, db *database.DB,

	logger *log.Logger, transCtx *model.TransferContext, _ any,

) error {

	if err := utils.JSONConvert(args, t); err != nil {

		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)

	}

	// Check condition: if set, skip if TransferInfo key is not "1"

	if t.Condition != "" {

		val, ok := transCtx.Transfer.TransferInfo[t.Condition]

		if !ok {

			logger.Debugf("SENDMESSAGE skipped: condition key %q not found in TransferInfo", t.Condition)

			return nil

		}

		if fmt.Sprint(val) != "1" {

			logger.Debugf("SENDMESSAGE skipped: condition key %q = %v (expected 1)", t.Condition, val)

			return nil

		}

	}

	// Resolve target partner: explicit arg > __replyPartner__ from TransferInfo.

	partnerName := t.Partner

	if partnerName == "" {

		if rp, ok := transCtx.Transfer.TransferInfo["__replyPartner__"]; ok {

			partnerName = fmt.Sprintf("%v", rp)

		}

	}

	if partnerName == "" {

		logger.Debug("SENDMESSAGE skipped: no target partner (no 'to' arg and no __replyPartner__ in TransferInfo)")

		return &WarningError{msg: "no ACK requested (no reply partner configured)"}

	}

	// Resolve account: explicit arg > __replyAccount__ > first account on partner.

	accountLogin := t.Account

	if accountLogin == "" {

		if ra, ok := transCtx.Transfer.TransferInfo["__replyAccount__"]; ok {

			accountLogin = fmt.Sprintf("%v", ra)

		}

	}

	var partner model.RemoteAgent

	if err := db.Get(&partner, "name=?", partnerName).Owner().
		Run(); database.IsNotFound(err) {

		return fmt.Errorf("%w: %q", ErrSendMessagePartnerNotFound, partnerName)

	} else if err != nil {

		return fmt.Errorf("failed to retrieve partner %q: %w", partnerName, err)

	}

	// Check protocol is PeSIT

	if partner.Protocol != "pesit" && partner.Protocol != "pesit-tls" {

		return fmt.Errorf("%w: partner %q uses protocol %q", ErrSendMessageNotPeSIT, partnerName, partner.Protocol)

	}

	// Resolve account

	var account model.RemoteAccount

	if accountLogin != "" {

		if err := db.Get(&account, "login=?", accountLogin).
			And("remote_agent_id=?", partner.ID).Run(); database.IsNotFound(err) {

			logger.Warningf("Account %q not found on partner %q, trying first available",

				accountLogin, partnerName)

		} else if err != nil {

			return fmt.Errorf("failed to retrieve account %q: %w", accountLogin, err)

		}

	}

	if account.ID == 0 {

		// Default: first account of the partner

		if err := db.Get(&account, "remote_agent_id=?", partner.ID).
			Run(); database.IsNotFound(err) {

			return fmt.Errorf("%w: no account found for partner %q", ErrSendMessageAccountNotFound, partnerName)

		} else if err != nil {

			return fmt.Errorf("failed to retrieve account for partner %q: %w", partnerName, err)

		}

	}

	logger.Infof("SENDMESSAGE: sending F.MESSAGE to partner %q as %q: %q",

		partnerName, account.Login, t.Message)

	if SendPeSITMessage == nil {

		return ErrSendMessageNotWired

	}

	// Parse transferID

	var tid uint32

	if t.TransferID != "" {

		tid64, convErr := strconv.ParseUint(t.TransferID, 10, 32)

		if convErr != nil {

			logger.Warningf("SENDMESSAGE: invalid transferId %q, using 0", t.TransferID)

		} else {

			tid = uint32(tid64)

		}

	}

	if err := SendPeSITMessage(db, partnerName, account.Login, tid, t.Message, logger); err != nil {

		return fmt.Errorf("SENDMESSAGE failed: %w", err)

	}

	logger.Infof("SENDMESSAGE: F.MESSAGE sent successfully to %s/%s", partnerName, account.Login)

	return nil

}
