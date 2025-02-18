package pesit

import (
	"context"
	"errors"
	"io"
	"path"
	"strconv"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const bytesPerKB int64 = 1024

type transferHandler struct {
	db      *database.DB
	logger  *log.Logger
	agent   *model.LocalAgent
	account *model.LocalAccount
	conf    *ServerConfig
	tracer  func() pipeline.Trace

	connFreetext string
	cftMode      bool

	ctx    context.Context
	cancel context.CancelCauseFunc
	pip    *pipeline.Pipeline
	file   *pipeline.FileStream
}

func (t *transferHandler) getRule(filepath string, isSend bool) (*model.Rule, error) {
	dir := path.Dir(filepath)

	rule, err := protoutils.GetClosestRule(t.db, t.logger, t.agent, t.account, dir, isSend)
	if err != nil {
		switch {
		case errors.Is(err, protoutils.ErrDatabase):
			return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "database error")
		case errors.Is(err, protoutils.ErrRuleNotFound):
			return nil, pesit.NewDiagnostic(pesit.CodeParameterError, "no rule found for filepath")
		case errors.Is(err, protoutils.ErrPermissionDenied):
			return nil, pesit.NewDiagnostic(pesit.CodeUnauthorizedCaller, "transfer refused: permission denied")
		default:
			return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "unexpected internal error")
		}
	}

	return rule, nil
}

func (t *transferHandler) getRuleByName(name string, isSend bool) (*model.Rule, error) {
	var rule model.Rule
	if err := t.db.Get(&rule, "name=? AND is_send=?", name, isSend); err != nil {
		t.logger.Error("Failed to retrieve rule: %v", err)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "database error")
	}

	return &rule, nil
}

//nolint:funlen //no easy way to split for now
func (t *transferHandler) SelectFile(req *pesit.ServerTransfer) error {
	// retrieve the rule and initialize the transfer
	var (
		trans     model.Transfer
		isSend    bool
		operation string
	)

	t.logger.Debug("Request received for file %q by %q", req.Filename(), req.ClientLogin())

	if t.conf.MaxMessageSize < req.MessageSize() {
		req.SetMessageSize(t.conf.MaxMessageSize)
	}

	if !req.IsSend() {
		if req.TransferID() == 0 {
			t.logger.Warning("Missing client transfer ID")

			return pesit.NewDiagnostic(pesit.CodeParameterError, "missing client transfer ID")
		}
	}

	switch {
	case req.IsSend():
		operation = "Download"
		isSend = true
	case req.IsReceive():
		operation = "Upload"
		isSend = false
		trans.SrcFilename = path.Base(req.Filename())
		trans.Filesize = model.UnknownSize
	default:
		t.logger.Warning("Unknown transfer method (should be either send or receive)")

		return pesit.NewDiagnostic(pesit.CodeMessageTypeRefused, "unknown transfer method")
	}

	var (
		rule    *model.Rule
		ruleErr error
	)

	if t.cftMode {
		rule, ruleErr = t.getRuleByName(req.FilenamePI12(), isSend)
	} else {
		rule, ruleErr = t.getRule(req.Filename(), isSend)
	}

	if ruleErr != nil {
		return ruleErr
	}

	trans.RemoteTransferID = utils.FormatUint(req.TransferID())
	trans.RuleID = rule.ID
	trans.LocalAccountID = utils.NewNullInt64(t.account.ID)

	if rule.IsSend {
		trans.SrcFilename = trimRequestPath(req.Filename(), rule)
	} else {
		trans.DestFilename = trimRequestPath(req.Filename(), rule)
	}

	// initialize the pipeline
	t.logger.Info("%s of file %q requested by %q using rule %q",
		operation, req.Filename(), req.ClientLogin(), rule.Name)

	t.ctx, t.cancel = context.WithCancelCause(context.Background())

	if err := t.initPipeline(req, &trans, rule); err != nil {
		t.logger.Warning("Transfer request for file %q refused: %v", req.Filename(), err)

		return err
	}

	req.StopReceived = stopReceived(t.pip)
	req.ConnectionAborted = connectionAborted(t.pip)
	req.RestartReceived = restartReceived(t.pip)
	req.CheckpointRequestReceived = checkpointRequestReceived(t.pip)

	if req.TransferID() == 0 {
		pesitID, convErr := strconv.ParseUint(trans.RemoteTransferID, 10, 32)
		if convErr != nil {
			t.pip.SetError(types.TeInternal, "failed to get parse Pesit transfer ID")
			t.logger.Error("Failed to parse Pesit transfer ID: %v", convErr)

			return pesit.NewDiagnostic(pesit.CodeInternalError, "failed to get parse Pesit transfer ID")
		}

		req.SetTransferID(uint32(pesitID))
	}

	t.logger.Info("Transfer request for file %q accepted", req.Filename())

	return nil
}

func (t *transferHandler) initPipeline(req *pesit.ServerTransfer,
	trans *model.Transfer, rule *model.Rule,
) error {
	if oldTrans, tErr := pipeline.GetOldTransfer(t.db, t.logger, trans); tErr != nil {
		t.logger.Error("Failed to check for existing transfers: %v", tErr)

		return transErrToPesitErr(tErr)
	} else {
		*trans = *oldTrans
	}

	pip, pipErr := pipeline.NewServerPipeline(t.db, t.logger, trans, snmp.GlobalService)
	if pipErr != nil {
		t.logger.Error("Failed to initialize pipeline: %v", pipErr)

		return transErrToPesitErr(pipErr)
	}

	pip.SetInterruptionHandlers(t.Pause, t.Interrupt, t.Cancel)
	t.pip = pip

	if t.tracer != nil {
		t.pip.Trace = t.tracer()
	}

	getFreetext(t.pip, clientConnFreetextKey, t.connFreetext)
	getFreetext(t.pip, clientTransFreetextKey, req.FreeText())

	return utils.RunWithCtx(t.ctx, func() error {
		// execute the pre-tasks
		if err := t.pip.PreTasks(); err != nil {
			return transErrToPesitErr(err)
		}

		// set the attributes of the response
		if rule.IsSend {
			stat, statErr := fs.Stat(t.pip.TransCtx.Transfer.LocalPath)
			if statErr != nil {
				return toPesitErr(pesit.CodeInternalError, statErr)
			}

			req.SetReservationSpace(makeReservationSpaceKB(stat), pesit.UnitKB)
			req.SetCreationDate(stat.ModTime())
		}

		return nil
	})
}

//nolint:gocritic //can't change the function's signature, this is an interface method
func (t *transferHandler) OpenFile(*pesit.ServerTransfer) error {
	t.pip.Logger.Debug("Opening file")

	// TODO handle compression once implemented in the library
	if err := utils.RunWithCtx(t.ctx, func() error {
		stream, stErr := t.pip.StartData()
		if stErr != nil {
			return transErrToPesitErr(stErr)
		}

		t.file = stream

		return nil
	}); err != nil {
		t.pip.Logger.Debug("File opening failed: %v", err)

		return err
	}

	t.pip.Logger.Debug("File opened successfully")

	return nil
}

func (t *transferHandler) StartDataTransfer(dtr *pesit.ServerTransfer) error {
	t.pip.Logger.Debug("Checking for recovery")

	if err := utils.RunWithCtx(t.ctx, func() error {
		// If the request is not a recovery, there is nothing to do
		if !dtr.IsRecovered() {
			return nil
		}

		// If the server is the receiver, set the recovery point
		if !t.pip.TransCtx.Rule.IsSend {
			recoveryPoint := uint32(t.pip.TransCtx.Transfer.Progress / int64(t.conf.CheckpointSize))
			dtr.SetRecoveryPoint(recoveryPoint)
		}

		// Then change the file offset to the corresponding byte
		offset := int64(dtr.RecoveryPoint()) * int64(t.conf.CheckpointSize)
		if _, err := t.file.Seek(offset, io.SeekStart); err != nil {
			return toPesitErr(pesit.CodeInternalError, err)
		}

		return nil
	}); err != nil {
		t.logger.Debug("Recovery check failed: %v", err)

		return err
	}

	t.pip.Logger.Debug("Recovery check successful")

	return nil
}

func (t *transferHandler) DataTransfer(trans *pesit.ServerTransfer) error {
	t.pip.Logger.Debug("Data transfer started")

	if err := utils.RunWithCtx(t.ctx, func() error {
		if t.pip.TransCtx.Rule.IsSend {
			if _, err := io.Copy(trans, t.file); err != nil {
				return toPesitErr(pesit.CodeInternalError, err)
			}
		} else {
			if _, err := io.Copy(t.file, trans); err != nil {
				return toPesitErr(pesit.CodeInternalError, err)
			}
		}

		return nil
	}); err != nil {
		t.pip.Logger.Debug("Data transfer failed: %v", err)

		var stopErr error

		switch {
		case errors.Is(err, errServerPause):
			stopErr = trans.Stop(pesit.StopSuspend, err)
		case errors.Is(err, errServerCanceled):
			stopErr = trans.Stop(pesit.StopCancel, err)
		default:
			stopErr = trans.Stop(pesit.StopError, err)
		}

		if stopErr != nil {
			t.pip.Logger.Error("Failed to stop transfer: %v", stopErr)
		}

		return err
	}

	t.pip.Logger.Debug("Data transfer completed")

	return nil
}

func (t *transferHandler) EndTransfer(_ *pesit.ServerTransfer, err error) error {
	if err != nil {
		t.pip.Logger.Warning("Data transfer ended with error: %v", err)
	} else {
		t.pip.Logger.Debug("Data transfer finished")
	}

	return nil
}

func (t *transferHandler) CloseFile(pErr error) error {
	t.pip.Logger.Debug("Closing file")

	if pErr != nil {
		t.handleError(pErr)

		return nil
	}

	if err := utils.RunWithCtx(t.ctx, func() error {
		if err := t.pip.EndData(); err != nil {
			return transErrToPesitErr(err)
		}

		return nil
	}); err != nil {
		t.pip.Logger.Debug("File closing failed: %v", err)

		return err
	}

	t.pip.Logger.Debug("File closed successfully")

	return nil
}

func (t *transferHandler) DeselectFile(pErr error) error {
	t.pip.Logger.Debug("Finalizing transfer")

	if pErr != nil {
		t.handleError(pErr)

		return nil
	}

	if err := utils.RunWithCtx(t.ctx, func() error {
		if err := t.pip.PostTasks(); err != nil {
			return transErrToPesitErr(err)
		}

		if err := t.pip.EndTransfer(); err != nil {
			return transErrToPesitErr(err)
		}

		return nil
	}); err != nil {
		t.pip.Logger.Debug("Transfer finalization failed: %v", err)

		return err
	}

	t.pip.Logger.Debug("Transfer finalization successful")

	return nil
}

var (
	errServerPause     = pesit.NewDiagnostic(pesit.CodeVolontaryTermination, "transfer paused by user")
	errServerCanceled  = pesit.NewDiagnostic(pesit.CodeVolontaryTermination, "transfer canceled by user")
	errServerInterrupt = pesit.NewDiagnostic(pesit.CodeUserServiceTermination, "service is shutting down")
)

func (t *transferHandler) stop(signal error) error {
	t.cancel(signal)

	return nil
}

func (t *transferHandler) Pause(context.Context) error {
	return t.stop(errServerPause)
}

func (t *transferHandler) Interrupt(context.Context) error {
	return t.stop(errServerInterrupt)
}

func (t *transferHandler) Cancel(context.Context) error {
	return t.stop(errServerCanceled)
}

func (t *transferHandler) handleError(err error) {
	var pErr pesit.Diagnostic
	errors.As(err, &pErr)

	switch pErr.GetCode() {
	case pesit.CodeVolontaryTermination:
		t.pip.Logger.Info("Transfer canceled by remote client")

		if err := t.pip.Cancel(t.ctx); err != nil {
			t.pip.Logger.Error("Failed to cancel transfer: %v", err)
		}
	case pesit.CodeTryLater:
		t.pip.Logger.Info("Transfer paused by remote client")

		if err := t.pip.Pause(t.ctx); err != nil {
			t.pip.Logger.Error("Failed to pause transfer: %v", err)
		}
	default:
		t.pip.Logger.Error("Error on remote client: %v", pErr)
		pipErr := pesitErrToPipErr("error on remote client", pErr)
		t.pip.SetError(pipErr.Code(), pipErr.Details())
	}
}
