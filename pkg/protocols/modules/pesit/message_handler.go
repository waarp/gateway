package pesit

import (
	"errors"
	"fmt"
	"time"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrDatabase = pesit.NewDiagnostic(pesit.CodeInternalError, "database error")

// HandleMessage implements pesit.MessageHandler. It is called when a remote
// partner sends a F.MESSAGE on an established connection. The message metadata
// is logged and persisted as TransferInfo on the referenced transfer (if found).
//
//nolint:gocritic //cannot change function signature
func (s *service) HandleMessage(_ *pesit.ServerConnection, msg pesit.MessageRequest) error {
	s.logger.Infof("F.MESSAGE received from %q: transferID=%d customerID=%q bankID=%q message=%q",
		msg.ClientLogin, msg.TransferID, msg.CustomerID, msg.BankID, msg.Message)

	if msg.TransferID == 0 {
		return nil
	}

	remoteID := utils.FormatUint(msg.TransferID)

	// Find the outgoing transfer that this ACK references.
	var outTrans model.NormalizedTransferView
	if err := s.db.Get(&outTrans, `remote_transfer_id=?`, remoteID).
		And("is_transfer=true").And("is_send=true").And("agent=?", msg.ClientLogin).
		OrderBy("start", false).Eager().Run(); database.IsNotFound(err) {
		s.logger.Debugf("No matching transfer found for F.MESSAGE transferID=%d", msg.TransferID)

		return nil
	} else if err != nil {
		s.logger.Debugf("Failed to retrieve transfer for F.MESSAGE transferID=%d: %v", msg.TransferID, err)

		return ErrDatabase
	}

	outTrans.TransferInfo[ackReceivedKey] = true
	outTrans.TransferInfo[ackReceivedOnKey] = time.Now().Format(time.RFC3339)

	if err := outTrans.UpdateInfo(s.db); err != nil {
		s.logger.Errorf("Failed to update transfer for F.MESSAGE: %v", err)

		return ErrDatabase
	}

	if err := s.relayMessage(&outTrans, &msg); err != nil {
		s.logger.Errorf("Failed to relay message: %v", err)

		return err
	}

	var realOutTrans model.Transfer
	if err := s.db.Get(&realOutTrans, "id=?", outTrans.ID).Eager().Run(); err != nil {
		s.logger.Errorf("Failed to retrieve transfer for F.MESSAGE transferID=%d: %v", msg.TransferID, err)

		return ErrDatabase
	}

	realOutTrans.Status = types.StatusDone
	delete(realOutTrans.TransferInfo, ackExpectedKey)
	if err := realOutTrans.MoveToHistory(s.db, s.logger, time.Now()); err != nil {
		s.logger.Errorf("Failed to move transfer to history: %v", err)

		return ErrDatabase
	}

	return nil
}

// relayMessage attempts to relay a F.MESSAGE upstream through the Store &
// Forward chain. It follows the __followID__ link to find the original
// incoming transfer, resolves the upstream partner, and sends the message.
func (s *service) relayMessage(outTrans *model.NormalizedTransferView, msg *pesit.MessageRequest) error {
	// Follow the chain: outgoing transfer (B→C) → __followID__ → incoming transfer (A→B)
	followID, idErr := utils.GetAs[uint64](outTrans.TransferInfo, model.FollowID)
	if idErr != nil {
		s.logger.Debug("No __followID__ on transfer, cannot relay F.MESSAGE upstream")

		return nil
	}

	var inTrans model.Transfer
	if err := s.db.Get(&inTrans, `id = (SELECT transfer_id FROM transfer_info WHERE
		name=? AND value=?)`, model.FollowID, followID).Eager().Run(); database.IsNotFound(err) {
		s.logger.Debugf("No upstream transfer found for followID %d", followID)

		return nil
	} else if err != nil {
		s.logger.Errorf("Failed to find incoming transfer: %v", err)

		return ErrDatabase
	}

	transferID, idErr := utils.ParseUint[uint32](inTrans.RemoteTransferID)
	if idErr != nil {
		s.logger.Errorf("Failed to parse remote transfer ID: %v", idErr)

		return ErrDatabase
	}

	if inTrans.IsServer() {
		if err := s.relayServerMessage(&inTrans, transferID, msg.Message); err != nil {
			return err
		}
	} else {
		transCtx, tErr := model.GetTransferContext(s.db, s.logger, &inTrans)
		if tErr != nil {
			s.logger.Errorf("Failed to get transfer context: %v", tErr)

			return ErrDatabase
		}

		if err := s.relayClientMessage(transCtx, transferID, msg.Message); err != nil {
			return err
		}
	}

	inTrans.Status = types.StatusDone
	inTrans.TransferInfo[ackSentKey] = true
	inTrans.TransferInfo[ackSentOnKey] = time.Now().Format(time.RFC3339)
	delete(inTrans.TransferInfo, ackExpectedKey)

	if err := inTrans.MoveToHistory(s.db, s.logger, time.Now()); err != nil {
		s.logger.Errorf("Failed to move transfer to history: %v", err)

		return ErrDatabase
	}

	return nil
}

func (s *service) relayClientMessage(transCtx *model.TransferContext,
	transferID uint32, message string,
) error {
	if err := sendMessage(s.db, s.logger, transCtx.RemoteAgent, transCtx.RemoteAccount,
		transCtx.RemoteAgentCreds, transCtx.RemoteAccountCreds, transCtx.Authorities,
		transferID, message); err != nil {
		s.logger.Errorf("Failed to send message: %v", err)

		if diag, isDiag := errors.AsType[pesit.Diagnostic](err); isDiag {
			return pesit.NewDiagnostic(diag.GetCode(),
				fmt.Sprintf("failed to relay message: %s", diag.GetMessage()))
		}

		return pesit.NewDiagnostic(pesit.CodeInternalError,
			fmt.Sprintf("failed to relay message: %v", err))
	}

	return nil
}

func (s *service) relayServerMessage(trans *model.Transfer, transferID uint32, message string,
) error {
	var inTrans model.NormalizedTransferView
	if err := s.db.Get(&inTrans, `id=?`, trans.ID).Run(); err != nil {
		s.logger.Errorf("Failed to retrieve transfer context: %v", err)

		return ErrDatabase
	}

	var partner model.RemoteAgent
	if err := s.db.Get(&partner, "name=?", inTrans.Account).Run(); err != nil {
		s.logger.Errorf("Failed to retrieve partner %q: %v", inTrans.Account, err)

		return ErrDatabase
	}

	account, dbErr := partner.GetAccount(s.db, inTrans.Agent)
	if dbErr != nil {
		s.logger.Errorf("Failed to retrieve account %q: %v", inTrans.Agent, dbErr)

		return ErrDatabase
	}

	if err := sendInitialMessage(s.db, s.logger, &partner, account,
		transferID, message); err != nil {
		s.logger.Errorf("Failed to send message: %v", err)

		if diag, isDiag := errors.AsType[pesit.Diagnostic](err); isDiag {
			return pesit.NewDiagnostic(diag.GetCode(),
				fmt.Sprintf("failed to relay message: %s", diag.GetMessage()))
		}

		return pesit.NewDiagnostic(pesit.CodeInternalError,
			fmt.Sprintf("failed to relay message: %v", err))
	}

	return nil
}
