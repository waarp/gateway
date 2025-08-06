// Package pipeline defines the transfer pipeline structure. This pipeline can
// (and should) be used to execute transfers, both by clients and servers.
// A pipeline can be initialized using the NewClientPipeline and NewServerPipeline
// functions.
package pipeline

import (
	"context"
	"fmt"
	"maps"
	"math"
	"reflect"
	"sync"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const TransferUpdateInterval = time.Second

// Pipeline is the structure regrouping all elements of the transfer Pipeline
// which are not protocol-dependent, such as task execution.
//
// A typical transfer's execution order should be as follows:
// - PreTasks,
// - StartData,
// - EndData,
// - PostTasks,
// - EndTransfer.
type Pipeline struct {
	DB       *database.DB
	TransCtx *model.TransferContext
	Logger   *log.Logger
	Stream   *FileStream
	Trace    Trace
	snmp     *snmp.Service

	machine          *statemachine.Machine
	interruption     interruption
	updTicker        *time.Ticker
	lastTransferInfo map[string]any

	storedErr *Error
	errOnce   sync.Once

	Runner *tasks.Runner
}

//nolint:funlen //function is fine as is
func newPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
	snmpService *snmp.Service,
) (*Pipeline, *Error) {
	pipeline := &Pipeline{
		DB:        db,
		Logger:    logger,
		TransCtx:  transCtx,
		snmp:      snmpService,
		machine:   pipelineSateMachine.New(),
		updTicker: time.NewTicker(TransferUpdateInterval),
		Runner:    tasks.NewTaskRunner(db, logger, transCtx),
	}

	if err := pipeline.setFilePaths(); err != nil {
		return nil, NewErrorWith(types.TeInternal, "failed to build the file paths", err)
	}

	if transCtx.Rule.IsSend {
		transCtx.Transfer.Filesize = getFilesize(transCtx.Transfer.LocalPath)
	}

	if transCtx.Transfer.Status == types.StatusAvailable {
		transCtx.Transfer.Start = time.Now()
	}

	if transCtx.Transfer.Status != types.StatusRunning {
		transCtx.Transfer.Status = types.StatusRunning
	}

	if transCtx.Transfer.Step < types.StepSetup {
		transCtx.Transfer.Step = types.StepSetup
	}

	if transCtx.Transfer.ErrCode != types.TeOk {
		transCtx.Transfer.ErrCode = types.TeOk
	}

	if transCtx.Transfer.ErrDetails != "" {
		transCtx.Transfer.ErrDetails = ""
	}

	if transCtx.Transfer.ID == 0 {
		if err := db.Insert(transCtx.Transfer).Run(); err != nil {
			logger.Errorf("failed to insert the new transfer entry: %v", err)

			return nil, NewErrorWith(types.TeInternal, "failed to insert the new transfer entry", err)
		}

		logger = logging.NewLogger(fmt.Sprintf("Pipeline %d (server)", transCtx.Transfer.ID))
		pipeline.Logger = logger
	} else if err := pipeline.UpdateTrans(); err != nil {
		logger.Errorf("Failed to update the transfer details: %v", err)

		return nil, NewErrorWith(types.TeInternal, "Failed to update the transfer details", err)
	}

	if err := List.add(pipeline); err != nil {
		logger.Errorf("Failed to add the pipeline to the list: %v", err)

		return nil, err
	}

	if analytics.GlobalService != nil {
		analytics.GlobalService.RunningTransfers.Add(1)
	}

	return pipeline, nil
}

func (p *Pipeline) SetInterruptionHandlers(
	pause, interrupt, cancel func(context.Context) error,
) {
	p.interruption = interruption{
		Pause:     pause,
		Interrupt: interrupt,
		Cancel:    cancel,
	}
}

func (p *Pipeline) forceUpdateTrans() *Error {
	if dbErr := p.DB.Update(p.TransCtx.Transfer).Run(); dbErr != nil {
		return NewErrorWith(types.TeInternal, "Failed to update transfer", dbErr)
	}

	return p.updateTransferInfo()
}

func (p *Pipeline) updateTransferInfo() *Error {
	if reflect.DeepEqual(p.TransCtx.TransInfo, p.lastTransferInfo) {
		return nil
	}

	if err := p.TransCtx.Transfer.SetTransferInfo(p.DB, p.TransCtx.TransInfo); err != nil {
		return p.internalError(types.TeInternal, "failed to update the transfer info", err)
	}

	p.lastTransferInfo = maps.Clone(p.TransCtx.TransInfo)

	return nil
}

// UpdateTrans updates the given columns of the pipeline's transfer.
func (p *Pipeline) UpdateTrans() *Error {
	select {
	case <-p.updTicker.C:
		if err := p.forceUpdateTrans(); err != nil {
			return p.internalErrorWithMsg(types.TeInternal, "Failed to update transfer",
				"database error", err)
		}
	default:
	}

	return nil
}

// PreTasks executes the transfer's pre-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
//
//nolint:dupl // factorizing would add complexity
func (p *Pipeline) PreTasks() *Error {
	if err := p.machine.Transition(statePreTasks); err != nil {
		return p.stateErr("PreTasks", p.machine.Current())
	}

	if p.TransCtx.Transfer.Step > types.StepPreTasks {
		if err := p.machine.Transition(statePreTasksDone); err != nil {
			return p.stateErr("PreTasksDone", p.machine.Current())
		}

		return nil
	}

	p.TransCtx.Transfer.Step = types.StepPreTasks
	if dbErr := p.UpdateTrans(); dbErr != nil {
		return dbErr
	}

	if err := p.Runner.PreTasks(p.Trace.OnPreTask); err != nil {
		return p.internalErrorWithMsg(err.Code, err.Details, "pre-tasks failed", err.Cause)
	}

	if err := p.machine.Transition(statePreTasksDone); err != nil {
		return p.stateErr("PreTasksDone", p.machine.Current())
	}

	return nil
}

// StartData marks the beginning of the data transfer. It opens the pipeline's
// TransferStream and returns it. In the case of a retry, the data transfer will
// resume at the offset indicated by the Transfer.Progress field.
func (p *Pipeline) StartData() (*FileStream, *Error) {
	if err := p.machine.Transition(stateDataStart); err != nil {
		return nil, p.stateErr("StartData", p.machine.Current())
	}

	isResume := false
	if p.TransCtx.Transfer.Step >= types.StepData {
		isResume = true
	} else {
		p.TransCtx.Transfer.Step = types.StepData
		p.TransCtx.Transfer.TaskNumber = 0

		if dbErr := p.UpdateTrans(); dbErr != nil {
			return nil, dbErr
		}
	}

	var fErr *Error
	if p.Stream, fErr = newFileStream(p, isResume); fErr != nil {
		return nil, fErr
	}

	if p.Trace.OnDataStart != nil {
		if err := p.Trace.OnDataStart(); err != nil {
			_ = p.Stream.file.Close() //nolint:errcheck //we don't care about the error here

			return nil, p.internalErrorWithMsg(types.TeInternal,
				"data start trace error", "failed to open file", err)
		}
	}

	exitState := utils.If(p.TransCtx.Rule.IsSend, stateReading, stateWriting)
	if err := p.machine.Transition(exitState); err != nil {
		_ = p.Stream.file.Close() //nolint:errcheck //we don't care about the error here

		return nil, p.stateErr("StartData", p.machine.Current())
	}

	return p.Stream, nil
}

// EndData marks the end of the data transfer. It closes the pipeline's
// TransferStream, and moves the file (when needed) from its temporary location
// to its final destination.
func (p *Pipeline) EndData() *Error {
	if err := p.machine.Transition(stateDataEnd); err != nil {
		return p.stateErr("EndData", p.machine.Current())
	}

	if err := p.Stream.close(); err != nil {
		return err
	}

	if err := p.Stream.move(); err != nil {
		return err
	}

	if p.TransCtx.Transfer.Filesize < 0 {
		if err := p.Stream.checkFileExist(); err != nil {
			return err
		}
	}

	if err := p.machine.Transition(stateDataEndDone); err != nil {
		return p.stateErr("EndDataDone", p.machine.Current())
	}

	return nil
}

// PostTasks executes the transfer's post-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
//
//nolint:dupl // factorizing would add complexity
func (p *Pipeline) PostTasks() *Error {
	if err := p.machine.Transition(statePostTasks); err != nil {
		return p.stateErr("PostTasks", p.machine.Current())
	}

	if p.TransCtx.Transfer.Step > types.StepPostTasks {
		if err := p.machine.Transition(statePostTasksDone); err != nil {
			return p.stateErr("PostTasksDone", p.machine.Current())
		}

		return nil
	}

	p.TransCtx.Transfer.Step = types.StepPostTasks
	if dbErr := p.UpdateTrans(); dbErr != nil {
		return dbErr
	}

	if err := p.Runner.PostTasks(p.Trace.OnPostTask); err != nil {
		return p.internalErrorWithMsg(err.Code, err.Details, "post-tasks failed", err.Cause)
	}

	if err := p.machine.Transition(statePostTasksDone); err != nil {
		return p.stateErr("PostTasksDone", p.machine.Current())
	}

	return nil
}

// EndTransfer signals that the transfer has ended normally, and archives in the
// transfer history.
func (p *Pipeline) EndTransfer() *Error {
	if err := p.machine.Transition(stateEndTransfer); err != nil {
		return p.stateErr("EndTransfer", p.machine.Current())
	}

	if p.Trace.OnFinalization != nil {
		if err := p.Trace.OnFinalization(); err != nil {
			return p.internalErrorWithMsg(types.TeInternal, "Test error on finalization",
				"error on transfer finalization", err)
		}
	}

	var sErr *Error

	p.errOnce.Do(func() {
		p.Runner.Stop()
		p.TransCtx.Transfer.Status = types.StatusDone
		p.TransCtx.Transfer.Step = types.StepNone
		p.TransCtx.Transfer.TaskNumber = 0

		if err := p.updateTransferInfo(); err != nil {
			p.Logger.Errorf("Failed to update transfer infos: %v", err)
			p.errorTasks()
			p.storedErr = sErr

			return
		}

		if err := p.TransCtx.Transfer.MoveToHistory(p.DB, p.Logger, time.Now()); err != nil {
			sErr = NewErrorWith(types.TeInternal, "Failed to move transfer to history", err)
			p.Logger.Errorf("Failed to move transfer to history: %v", err)

			if mErr := p.machine.Transition(stateError); mErr != nil {
				p.Logger.Warningf(`Failed to transition to "error" state: %v`, mErr)
			}

			p.errorTasks()
			p.storedErr = sErr

			return
		}

		p.doneOK()
	})

	return sErr
}

func (p *Pipeline) errorTasks() {
	oldStep := p.TransCtx.Transfer.Step
	oldTask := p.TransCtx.Transfer.TaskNumber

	defer func() {
		p.TransCtx.Transfer.Step = oldStep
		p.TransCtx.Transfer.TaskNumber = oldTask

		if dbErr := p.forceUpdateTrans(); dbErr != nil {
			p.Logger.Errorf("Failed to reset transfer step after error-tasks: %v", dbErr)
		}
	}()

	p.TransCtx.Transfer.TaskNumber = 0
	p.TransCtx.Transfer.Step = types.StepErrorTasks

	if dbErr := p.forceUpdateTrans(); dbErr != nil {
		p.Logger.Errorf("Failed to update transfer step for error-tasks: %v", dbErr)
	}

	if err := p.Runner.ErrorTasks(p.Trace.OnErrorTask); err != nil {
		p.Logger.Errorf("Error-tasks failed: %s", err.Details)
	}
}

// SetError stops the pipeline and sets its error to the given value.
func (p *Pipeline) SetError(code types.TransferErrorCode, details string) {
	p.externalError(code, details)
}

func (p *Pipeline) stop() {
	switch p.machine.Last() {
	case statePreTasks, statePostTasks:
		p.Runner.Stop()
	case stateReading, stateWriting, stateDataEnd:
		p.Stream.stop()
	}
}

// Pause stops the pipeline and pauses the transfer.
func (p *Pipeline) Pause(ctx context.Context) error {
	if p.Trace.OnPause != nil {
		p.Trace.OnPause()
	}

	return p.halt(ctx, errPause, types.StatusPaused, p.interruption.Pause)
}

// Interrupt stops the pipeline and interrupts the transfer.
func (p *Pipeline) Interrupt(ctx context.Context) error {
	if p.Trace.OnInterruption != nil {
		p.Trace.OnInterruption()
	}

	return p.halt(ctx, errInterrupted, types.StatusInterrupted, p.interruption.Interrupt)
}

func (p *Pipeline) halt(ctx context.Context, err *Error, status types.TransferStatus,
	handle func(context.Context) error,
) error {
	var haltErr error

	p.errOnce.Do(func() {
		p.Logger.Info(err.details)
		p.storedErr = err

		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
			haltErr = fmt.Errorf("failed to transition to 'error' state: %w", mErr)
		}

		if handle != nil {
			if err := handle(ctx); err != nil {
				haltErr = err
			}
		}

		p.stop()
		p.doneErr(status)
	})

	return haltErr
}

// Cancel stops the pipeline and cancels the transfer.
func (p *Pipeline) Cancel(ctx context.Context) error {
	var cancelErr error

	p.errOnce.Do(func() {
		p.Logger.Info("Transfer canceled by user")
		p.storedErr = errCanceled

		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
			cancelErr = fmt.Errorf("failed to transition to 'error' state: %w", mErr)
		}

		if p.Trace.OnCancel != nil {
			p.Trace.OnCancel()
		}

		if p.interruption.Cancel != nil {
			if err := p.interruption.Cancel(ctx); err != nil {
				p.Logger.Warningf("Failed to cancel transfer: %v", err)
				cancelErr = fmt.Errorf("failed to cancel transfer: %w", err)
			}
		}

		p.stop()

		p.TransCtx.Transfer.Status = types.StatusCancelled
		if err := p.TransCtx.Transfer.MoveToHistory(p.DB, p.Logger, time.Now()); err != nil {
			p.Logger.Errorf("Failed to move canceled transfer to history: %v", err)
			cancelErr = fmt.Errorf("failed to move canceled transfer to history: %w", err)
		}

		p.done(stateInError)
	})

	return cancelErr
}

// RebuildFilepaths rebuilds the transfer's local and remote paths, just like it
// is done at the beginning of the transfer. This is useful if the file's name
// has changed during the transfer.
func (p *Pipeline) RebuildFilepaths(newFile string) *Error {
	var srcFile, dstFile string

	switch {
	case !p.TransCtx.Transfer.IsServer():
		srcFile, dstFile = newFile, newFile
	case p.TransCtx.Rule.IsSend:
		srcFile = newFile
	default:
		dstFile = newFile
	}

	p.TransCtx.Transfer.LocalPath = ""
	p.TransCtx.Transfer.RemotePath = ""

	if err := p.setCustomFilePaths(srcFile, dstFile); err != nil {
		return p.internalErrorWithMsg(types.TeInternal, "failed to rebuild the paths",
			"filesystem error", err)
	}

	return p.UpdateTrans()
}

func (p *Pipeline) doneErr(status types.TransferStatus) {
	defer func() {
		p.Logger.Debugf("Transfer ended in error in %v",
			time.Since(p.TransCtx.Transfer.Start))
	}()

	p.TransCtx.Transfer.Status = status

	if p.TransCtx.Transfer.RemainingTries > 0 {
		p.TransCtx.Transfer.NextRetry = time.Now().UTC().
			Add(time.Second * time.Duration(p.TransCtx.Transfer.NextRetryDelay))
		incrementRetryDelay(p.TransCtx.Transfer)
	}

	if err := p.forceUpdateTrans(); err != nil {
		p.Logger.Errorf("Failed to update transfer status to %v: %v", status, err)
	}

	if p.snmp != nil {
		p.snmp.ReportTransferError(p.TransCtx.Transfer.ID)
	}

	p.done(stateInError)
}

func (p *Pipeline) doneOK() {
	defer func() {
		p.Logger.Infof("Transfer ended without errors in %s",
			time.Since(p.TransCtx.Transfer.Start))
	}()

	p.done(stateAllDone)
}

func (p *Pipeline) done(state statemachine.State) {
	if mErr := p.machine.Transition(state); mErr != nil {
		p.Logger.Warningf("Failed to transition to %q state: %v", state, mErr)
	}

	if p.Trace.OnTransferEnd != nil {
		p.Trace.OnTransferEnd()
	}

	List.remove(p.TransCtx.Transfer.ID)

	if analytics.GlobalService != nil {
		analytics.GlobalService.RunningTransfers.Add(-1)
	}
}

func (p *Pipeline) IsServer() bool { return p.TransCtx.Transfer.IsServer() }
func (p *Pipeline) IsSend() bool   { return p.TransCtx.Rule.IsSend }

func incrementRetryDelay(trans *model.Transfer) {
	newDelay := float32(trans.NextRetryDelay) * trans.RetryIncrementFactor
	if newDelay > float32(math.MaxInt32) {
		trans.NextRetryDelay = math.MaxInt32
	} else {
		trans.NextRetryDelay = int32(newDelay)
	}
}
