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
	if err := t.db.Get(&rule, "name=? AND is_send=?", name, isSend).Run(); err != nil {
		t.logger.Errorf("Failed to retrieve rule: %v", err)

		return nil, pesit.NewDiagnostic(pesit.CodeInternalError, "database error")
	}

	return &rule, nil
}

//nolint:funlen //no easy way to split for now
func (t *transferHandler) SelectFile(req *pesit.ServerTransfer) error {
	// retrieve the rule and initialize the transfer
	var (
		isSend    bool
		operation string
	)

	t.logger.Debugf("Request received for file %q by %q", req.Filename(), req.ClientLogin())

	req.SetArticleSize(defaultArticleSize)

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
	default:
		t.logger.Warning("Unknown transfer method (should be either send or receive)")

		return pesit.NewDiagnostic(pesit.CodeMessageTypeRefused, "unknown transfer method")
	}

	var (
		rule             *model.Rule
		ruleErr          error
		remoteTransferID string
	)

	if t.cftMode {
		rule, ruleErr = t.getRuleByName(req.FilenamePI12(), isSend)
	} else {
		rule, ruleErr = t.getRule(req.Filename(), isSend)
	}

	if ruleErr != nil {
		return ruleErr
	}

	filepath := trimRequestPath(req.Filename(), rule)

	if req.TransferID() != 0 {
		remoteTransferID = utils.FormatUint(req.TransferID())
	}

	// initialize the pipeline
	t.logger.Infof("%s of file %q requested by %q using rule %q",
		operation, req.Filename(), req.ClientLogin(), rule.Name)

	t.ctx, t.cancel = context.WithCancelCause(context.Background())

	if err := t.initPipeline(req, remoteTransferID, filepath, rule); err != nil {
		t.logger.Warningf("Transfer request for file %q refused: %v", req.Filename(), err)

		return err
	}

	req.StopReceived = stopReceived(t.pip)
	req.ConnectionAborted = connectionAborted(t.pip)
	req.RestartReceived = restartReceived(t.pip)
	req.CheckpointRequestReceived = checkpointRequestReceived(t.pip)

	if req.TransferID() == 0 {
		pesitID, convErr := strconv.ParseUint(t.pip.TransCtx.Transfer.RemoteTransferID, 10, 32)
		if convErr != nil {
			t.pip.SetError(types.TeInternal, "failed to parse Pesit transfer ID")
			t.logger.Errorf("Failed to parse Pesit transfer ID: %v", convErr)

			return pesit.NewDiagnostic(pesit.CodeInternalError, "failed to parse Pesit transfer ID")
		}

		req.SetTransferID(uint32(pesitID))
	}

	t.logger.Infof("Transfer request for file %q accepted", req.Filename())

	return nil
}

func (t *transferHandler) mkTransfer(remoteID, filepath string, rule *model.Rule,
) (*model.Transfer, error) {
	if trans, err := pipeline.GetOldTransferByRemoteID(t.db, remoteID, t.account,
		rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		return nil, transErrToPesitErr(err)
	}

	// CFT mode -> no filename, so we use the rule instead
	if filepath == "" {
		if rule.IsSend {
			if trans, err := pipeline.GetAvailableTransferByRule(t.db, remoteID, t.account, rule); err == nil {
				return trans, nil
			} else if !database.IsNotFound(err) {
				return nil, transErrToPesitErr(err)
			}

			return nil, pesit.NewDiagnostic(pesit.CodeFileNotExists, "no available transfer found")
		}

		filepath = generateDestFilename(remoteID, t.account, rule)
	}

	if trans, err := pipeline.GetAvailableTransferByFilename(t.db, filepath, remoteID,
		t.account, rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		return nil, transErrToPesitErr(err)
	}

	return pipeline.MakeServerTransfer(remoteID, filepath, t.account, rule), nil
}

func (t *transferHandler) initPipeline(req *pesit.ServerTransfer,
	remoteID, filepath string, rule *model.Rule,
) error {
	trans, tErr := t.mkTransfer(remoteID, filepath, rule)
	if tErr != nil {
		t.logger.Errorf("Failed to check for existing transfers: %v", tErr)

		return tErr
	}

	pip, pipErr := pipeline.NewServerPipeline(t.db, t.logger, trans, snmp.GlobalService)
	if pipErr != nil {
		t.logger.Errorf("Failed to initialize pipeline: %v", pipErr)

		return transErrToPesitErr(pipErr)
	}

	pip.SetInterruptionHandlers(t.Pause, t.Interrupt, t.Cancel)
	t.pip = pip

	if t.tracer != nil {
		t.pip.Trace = t.tracer()
	}

	setTransInfo(t.pip, clientConnFreetextKey, t.connFreetext)
	setTransInfo(t.pip, clientTransFreetextKey, req.FreeText())
	setTransInfo(t.pip, customerIDKey, req.CustomerID())
	setTransInfo(t.pip, bankIDKey, req.BankID())

	if pip.TransCtx.Rule.IsSend {
		if err := setFileType(t.pip, req); err != nil {
			return err
		}

		if err := setFileOrganization(t.pip, req); err != nil {
			return nil
		}

		if err := setFileEncoding(t.pip, req); err != nil {
			return err
		}
	} else {
		setTransInfo(t.pip, fileEncodingKey, req.DataCoding().String())
		setTransInfo(t.pip, fileTypeKey, req.FileType())
		setTransInfo(t.pip, organizationKey, req.FileOrganization().String())
	}

	if t.pip.TransCtx.Rule.IsSend {
		if err := setFreetext(pip, serverTransFreetextKey, req); err != nil {
			t.logger.Errorf("Failed to set server transfer freetext: %v", err)

			return transErrToPesitErr(err)
		}
	}

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
		t.pip.Logger.Debugf("File opening failed: %v", err)

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
		t.logger.Debugf("Recovery check failed: %v", err)

		return err
	}

	t.pip.Logger.Debug("Recovery check successful")

	return nil
}

func (t *transferHandler) receiveTransfer(trans *pesit.ServerTransfer) error {
	var articlesLen []int64

	for {
		article, aErr := trans.GetNextRecvArticle()
		if errors.Is(aErr, pesit.ErrNoMoreArticle) {
			return nil
		} else if aErr != nil {
			t.handleError(aErr)

			//nolint:wrapcheck //wrapping adds nothing here
			return aErr
		}

		start := t.pip.TransCtx.Transfer.Progress

		if _, err := io.Copy(t.file, article); err != nil {
			return toPesitErr(pesit.CodeInternalError, err)
		}

		end := t.pip.TransCtx.Transfer.Progress
		articlesLen = append(articlesLen, end-start)
		t.pip.TransCtx.TransInfo[articlesLengthsKey] = articlesLen
	}
}

func (t *transferHandler) sendTransfer(trans *pesit.ServerTransfer) error {
	lengths, isMArticles := isMultiArticles(t.pip)
	if !isMArticles {
		if _, err := io.Copy(trans, t.file); err != nil {
			return toPesitErr(pesit.CodeInternalError, err)
		}

		return nil
	}

	trans.SetArticleFormat(pesit.FormatVariable)
	trans.SetManualArticleHandling(true)

	for _, length := range lengths {
		article, aErr := trans.StartNextSendArticle()
		if aErr != nil {
			t.handleError(aErr)

			return aErr //nolint:wrapcheck //wrapping adds nothing here
		}

		file := io.LimitReader(t.file, length)

		if _, err := io.Copy(article, file); err != nil {
			return toPesitErr(pesit.CodeInternalError, err)
		}
	}

	return nil
}

func (t *transferHandler) DataTransfer(trans *pesit.ServerTransfer) error {
	t.pip.Logger.Debug("Data transfer started")

	if err := utils.RunWithCtx(t.ctx, func() error {
		if t.pip.TransCtx.Rule.IsSend {
			return t.sendTransfer(trans)
		}

		return t.receiveTransfer(trans)
	}); err != nil {
		t.pip.Logger.Debugf("Data transfer failed: %v", err)

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
			t.pip.Logger.Errorf("Failed to stop transfer: %v", stopErr)
		}

		return err
	}

	t.pip.Logger.Debug("Data transfer completed")

	return nil
}

func (t *transferHandler) EndTransfer(_ *pesit.ServerTransfer, err error) error {
	if err != nil {
		t.pip.Logger.Warningf("Data transfer ended with error: %v", err)
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
		t.pip.Logger.Debugf("File closing failed: %v", err)

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
		t.pip.Logger.Debugf("Transfer finalization failed: %v", err)

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
	var pesitErr pesit.Diagnostic
	errors.As(err, &pesitErr)

	switch pesitErr.GetCode() {
	case pesit.CodeVolontaryTermination:
		t.pip.Logger.Info("Transfer canceled by remote client")

		if cErr := t.pip.Cancel(t.ctx); cErr != nil {
			t.pip.Logger.Errorf("Failed to cancel transfer: %v", cErr)
		}
	case pesit.CodeTryLater:
		t.pip.Logger.Info("Transfer paused by remote client")

		if pErr := t.pip.Pause(t.ctx); pErr != nil {
			t.pip.Logger.Errorf("Failed to pause transfer: %v", pErr)
		}
	default:
		t.pip.Logger.Errorf("Error on remote client: %v", pesitErr)
		pipErr := pesitErrToPipErr("error on remote client", pesitErr)
		t.pip.SetError(pipErr.Code(), pipErr.Details())
	}
}
