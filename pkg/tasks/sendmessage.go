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
	ErrSendMessageNoPartner       = errors.New(`missing "partner" argument`)
	ErrSendMessagePartnerNotFound = errors.New("sendmessage partner not found")
	ErrSendMessageAccountNotFound = errors.New("sendmessage account not found")
	ErrSendMessageNotPeSIT        = errors.New("sendmessage only works with PeSIT partners")
	ErrSendMessageNotWired        = errors.New("SENDMESSAGE not available: PeSIT module not loaded")
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
//	partner   (required) Remote partner name. Supports variable substitution.
//	account   (optional) Remote account login. Defaults to first account of the partner.
//	message   (optional) Message content. Supports variable substitution. Max 4096 chars.
//	transferId (optional) Transfer ID to reference in the F.MESSAGE. Supports variables.
//	condition (optional) TransferInfo key to check. If set and value != "1", skip.
type sendMessageTask struct {
	Partner    string `json:"partner"`
	Account    string `json:"account"`
	Message    string `json:"message"`
	TransferID string `json:"transferId"`
	Condition  string `json:"condition"`
}

func (t *sendMessageTask) Validate(args map[string]string) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse sendmessage arguments: %w", err)
	}

	if t.Partner == "" {
		return ErrSendMessageNoPartner
	}

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

	// Resolve partner
	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", t.Partner).Owner().
		Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w: %q", ErrSendMessagePartnerNotFound, t.Partner)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve partner %q: %w", t.Partner, err)
	}

	// Check protocol is PeSIT
	if partner.Protocol != "pesit" && partner.Protocol != "pesit-tls" {
		return fmt.Errorf("%w: partner %q uses protocol %q", ErrSendMessageNotPeSIT, t.Partner, partner.Protocol)
	}

	// Resolve account
	var account model.RemoteAccount
	if t.Account != "" {
		if err := db.Get(&account, "login=?", t.Account).
			And("remote_agent_id=?", partner.ID).Run(); database.IsNotFound(err) {
			return fmt.Errorf("%w: %q", ErrSendMessageAccountNotFound, t.Account)
		} else if err != nil {
			return fmt.Errorf("failed to retrieve account %q: %w", t.Account, err)
		}
	} else {
		// Default: first account of the partner
		if err := db.Get(&account, "remote_agent_id=?", partner.ID).
			Run(); database.IsNotFound(err) {
			return fmt.Errorf("%w: no account found for partner %q", ErrSendMessageAccountNotFound, t.Partner)
		} else if err != nil {
			return fmt.Errorf("failed to retrieve account for partner %q: %w", t.Partner, err)
		}
	}

	logger.Infof("SENDMESSAGE: sending F.MESSAGE to partner %q as %q: %q",
		t.Partner, account.Login, t.Message)

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

	if err := SendPeSITMessage(db, t.Partner, account.Login, tid, t.Message, logger); err != nil {
		return fmt.Errorf("SENDMESSAGE failed: %w", err)
	}

	logger.Infof("SENDMESSAGE: F.MESSAGE sent successfully to %s/%s", t.Partner, account.Login)

	return nil
}
