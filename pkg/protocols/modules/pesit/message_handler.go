package pesit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrDatabase   = pesit.NewDiagnostic(pesit.CodeInternalError, "database error")
	ErrACKRunning = pesit.NewDiagnostic(pesit.CodeParameterError, "cannot acknowledge a running transfer")
)

var _ pesit.MessageHandler = &service{}

// HandleMessage implements pesit.MessageHandler. It is called when a remote
// partner sends a F.MESSAGE on an established connection. The message metadata
// is logged and persisted as TransferInfo on the referenced transfer (if found).
//
//nolint:gocritic //cannot change function signature
func (s *service) HandleMessage(req *pesit.MessageRequest, cont io.Reader,
) (*pesit.MessageResult, error) {
	receivedTime := time.Now()

	content, rErr := io.ReadAll(cont)
	if rErr != nil {
		s.logger.Errorf("Failed to read message: %s", rErr)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "failed to read message")
	}

	s.logger.Infof("F.MESSAGE received from %q: transferID=%d customerID=%q bankID=%q message=%q",
		req.ClientLogin, req.TransferID, req.CustomerID, req.BankID, string(content))

	if req.MessageType != pesit.FileACK || req.TransferID == 0 {
		return &pesit.MessageResult{}, nil
	}

	partner, partErr := s.findPartnerByLogin(req.ClientLogin)
	if partErr != nil {
		return nil, partErr
	}

	remoteID := utils.FormatUint(req.TransferID)

	// Find the outgoing transfer that this ACK references.
	var outTrans model.NormalizedTransferView
	if err := s.db.Get(&outTrans, `remote_transfer_id=?`, remoteID).
		And("is_transfer=true").
		And("is_send=true").
		And("agent=?", partner.Name).
		OrderBy("start", false).Eager().Run(); database.IsNotFound(err) {
		s.logger.Debugf("No matching transfer found for F.MESSAGE transferID=%d", req.TransferID)

		return &pesit.MessageResult{}, nil
	} else if err != nil {
		s.logger.Debugf("Failed to retrieve transfer for F.MESSAGE transferID=%d: %v", req.TransferID, err)

		return nil, ErrDatabase
	}

	if err := s.relayMessage(&outTrans, bytes.NewReader(content)); err != nil {
		s.logger.Errorf("Failed to relay message: %v", err)

		return &pesit.MessageResult{}, err
	}

	outTrans.TransferInfo[ackReceivedKey] = true
	outTrans.TransferInfo[ackReceivedOnKey] = receivedTime
	delete(outTrans.TransferInfo, ackExpectedKey)
	delete(outTrans.TransferInfo, ackWaitSinceKey)

	if err := outTrans.UpdateInfo(s.db); err != nil {
		s.logger.Errorf("Failed to update transfer for F.MESSAGE: %v", err)

		return nil, ErrDatabase
	}

	return &pesit.MessageResult{}, nil
}

func (s *service) findPartnerByLogin(login string) (*model.RemoteAgent, error) {
	var pesitPartners model.RemoteAgents
	if err := s.db.Select(&pesitPartners).Run(); err != nil {
		s.logger.Errorf("Failed to retrieve remote agents: %v", err)

		return nil, ErrDatabase
	}

	for _, partner := range pesitPartners {
		if partner.Name == login || partner.ProtoConfig["login"] == login {
			return partner, nil
		}
	}

	s.logger.Warningf("No partner found for login %q", login)

	return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller, "no partner found for login")
}

// relayMessage attempts to relay a F.MESSAGE upstream through the Store &
// Forward chain. It follows the __followID__ link to find the original
// incoming transfer, resolves the upstream partner, and sends the message.
func (s *service) relayMessage(outTrans *model.NormalizedTransferView,
	message io.Reader,
) error {
	// Follow the chain: outgoing transfer (B→C) → __followID__ → incoming transfer (A→B)
	followID, idErr := utils.GetAs[uint64](outTrans.TransferInfo, model.FollowID)
	if idErr != nil {
		s.logger.Debug("No __followID__ on transfer, cannot relay F.MESSAGE upstream")

		return nil
	}

	var infos model.Slice[model.NormalizedTransferInfo]
	if err := s.db.Select(&infos).Where("name=?", model.FollowID).
		Where("value=?", followID).Run(); err != nil {
		s.logger.Errorf("Failed to find followID: %v", err)

		return ErrDatabase
	}

	var inTrans model.NormalizedTransferView
	for _, info := range infos {
		if info.OwnerID == outTrans.ID {
			continue
		}

		if err := s.db.Get(&inTrans, "id=?", info.OwnerID).Run(); err != nil {
			s.logger.Errorf("Failed to retrieve transfer: %v", err)

			return ErrDatabase
		}

		break
	}

	transferID, idErr := utils.ParseUint[uint32](inTrans.RemoteTransferID)
	if idErr != nil {
		s.logger.Errorf("Failed to parse remote transfer ID: %v", idErr)

		return ErrDatabase
	}

	if inTrans.IsServer {
		if err := s.relayServerMessage(&inTrans, transferID, message); err != nil {
			return err
		}
	} else {
		transCtx, tErr := model.GetHistoryContext(s.db, s.logger, &inTrans)
		if tErr != nil {
			s.logger.Errorf("Failed to get transfer context: %v", tErr)

			return ErrDatabase
		}

		if err := s.relayClientMessage(transCtx, transferID, message); err != nil {
			return err
		}
	}

	inTrans.TransferInfo[ackSentKey] = true
	inTrans.TransferInfo[ackSentOnKey] = time.Now().Format(time.RFC3339)

	if err := inTrans.UpdateInfo(s.db); err != nil {
		s.logger.Errorf("Failed to update transfer info: %v", err)

		return ErrDatabase
	}

	return nil
}

func (s *service) relayClientMessage(transCtx *model.TransferContext,
	transferID uint32, message io.Reader,
) error {
	filename := transCtx.Transfer.SrcFilename
	if filename == "" {
		filename = transCtx.Transfer.DestFilename
	}

	if err := sendMessage(s.db, s.logger, transCtx.RemoteAgent, transCtx.RemoteAccount,
		transCtx.RemoteAgentCreds, transCtx.RemoteAccountCreds, transCtx.Authorities,
		transCtx.Transfer.TransferInfo, transferID, filename, message); err != nil {
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

func (s *service) relayServerMessage(inTrans *model.NormalizedTransferView, transferID uint32,
	message io.Reader,
) error {
	partner, partErr := s.findPartnerByLogin(inTrans.Account)
	if partErr != nil {
		s.logger.Errorf("Failed to retrieve partner %q: %v", inTrans.Account, partErr)

		return ErrDatabase
	}

	account, dbErr := partner.GetAccount(s.db, inTrans.Agent)
	if dbErr != nil {
		s.logger.Errorf("Failed to retrieve account %q: %v", inTrans.Agent, dbErr)

		return ErrDatabase
	}

	filename := inTrans.SrcFilename
	if filename == "" {
		filename = inTrans.DestFilename
	}

	if err := sendInitialMessage(s.db, s.logger, partner, account,
		inTrans.TransferInfo, transferID, filename, message); err != nil {
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
