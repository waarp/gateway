package pipeline

import (
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
)

var (
	errDatabase     = types.NewTransferError(types.TeInternal, "database error")
	errStateMachine = types.NewTransferError(types.TeInternal, "internal transfer error")
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
	Stream   TransferStream

	machine   *statemachine.Machine
	updTicker *time.Ticker
	errOnce   sync.Once

	runner *tasks.Runner
}

func newPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) (*Pipeline, *types.TransferError) {
	pipeline := &Pipeline{
		DB:        db,
		Logger:    logger,
		TransCtx:  transCtx,
		machine:   pipelineSateMachine.New(),
		updTicker: time.NewTicker(TransferUpdateInterval),
		runner:    tasks.NewTaskRunner(db, logger, transCtx),
	}

	pipeline.setFilePaths()

	if transCtx.Rule.IsSend {
		if err := pipeline.checkFileExist(); err != nil {
			return nil, err
		}
	}

	if transCtx.Transfer.Status != types.StatusRunning {
		transCtx.Transfer.Status = types.StatusRunning
	}

	if transCtx.Transfer.Step < types.StepSetup {
		transCtx.Transfer.Step = types.StepSetup
	}

	if transCtx.Transfer.Error.Code != types.TeOk {
		transCtx.Transfer.Error.Code = types.TeOk
	}

	if transCtx.Transfer.Error.Details != "" {
		transCtx.Transfer.Error.Details = ""
	}

	return pipeline, nil
}

func (p *Pipeline) init() (err *types.TransferError) {
	if p.TransCtx.Rule.IsSend {
		if !TransferOutCount.Add() {
			err = errLimitReached
		}
	} else {
		if !TransferInCount.Add() {
			err = errLimitReached
		}
	}

	if err != nil {
		p.Logger.Error("Failed to initiate transfer pipeline: ¨%s", err)
		p.TransCtx.Transfer.Error = *err

		if dbErr := p.UpdateTrans(); dbErr != nil {
			return dbErr
		}
	}

	return err
}

// UpdateTrans updates the given columns of the pipeline's transfer.
func (p *Pipeline) UpdateTrans() *types.TransferError {
	return p.doUpdateTrans(p.handleError)
}

func (p *Pipeline) doUpdateTrans(handleError func(types.TransferErrorCode,
	string, string),
) *types.TransferError {
	select {
	case <-p.updTicker.C:
		if dbErr := p.DB.Update(p.TransCtx.Transfer).Run(); dbErr != nil {
			handleError(types.TeInternal, "Failed to update transfer",
				dbErr.Error())

			return errDatabase
		}
	default:
	}

	stage := DataWrite
	if p.TransCtx.Rule.IsSend {
		stage = DataRead
	}

	if testErr := Tester.getError(stage, p.TransCtx.Transfer.Progress); testErr != nil {
		p.handleError(testErr.Code, "test error", testErr.Details)

		return testErr
	}

	return nil
}

// PreTasks executes the transfer's pre-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
//
//nolint:dupl // factorizing would add complexity
func (p *Pipeline) PreTasks() *types.TransferError {
	defer Tester.preTasksDone(p.TransCtx.Transfer)

	if err := p.machine.Transition(statePreTasks); err != nil {
		p.handleStateErr("PreTasks", p.machine.Current())

		return errStateMachine
	}

	if p.TransCtx.Transfer.Step > types.StepPreTasks {
		if err := p.machine.Transition(statePreTasksDone); err != nil {
			p.handleStateErr("PreTasksDone", p.machine.Current())

			return errStateMachine
		}

		return nil
	}

	p.TransCtx.Transfer.Step = types.StepPreTasks
	if dbErr := p.UpdateTrans(); dbErr != nil {
		return dbErr
	}

	if err := p.runner.PreTasks(); err != nil {
		p.handleError(err.Code, "Pre-tasks failed", err.Details)

		return types.NewTransferError(types.TeExternalOperation, "pre-tasks failed")
	}

	if err := p.machine.Transition(statePreTasksDone); err != nil {
		p.handleStateErr("PreTasksDone", p.machine.Current())

		return errStateMachine
	}

	return nil
}

// StartData marks the beginning of the data transfer. It opens the pipeline's
// TransferStream and returns it. In the case of a retry, the data transfer will
// resume at the offset indicated by the Transfer.Progress field.
func (p *Pipeline) StartData() (TransferStream, *types.TransferError) {
	if err := p.machine.Transition(stateDataStart); err != nil {
		p.handleStateErr("StartData", p.machine.Current())

		return nil, errStateMachine
	}

	transitionData := func() *types.TransferError {
		newState := stateWriting
		if p.TransCtx.Rule.IsSend {
			newState = stateReading
		}

		if err := p.machine.Transition(newState); err != nil {
			p.handleStateErr("StartData", p.machine.Current())

			return errStateMachine
		}

		return nil
	}

	if p.TransCtx.Transfer.Step > types.StepData {
		p.Stream = newVoidStream(p)

		if err := transitionData(); err != nil {
			return nil, err
		}

		return p.Stream, nil
	}

	isResume := false
	if p.TransCtx.Transfer.Step == types.StepData {
		isResume = true
	} else {
		p.TransCtx.Transfer.Step = types.StepData
		p.TransCtx.Transfer.TaskNumber = 0

		if dbErr := p.UpdateTrans(); dbErr != nil {
			return nil, dbErr
		}
	}

	var tErr *types.TransferError

	p.Stream, tErr = newFileStream(p, isResume)
	if tErr != nil {
		p.handleError(tErr.Code, "Failed to create file stream", tErr.Details)

		return nil, tErr
	}

	if err := transitionData(); err != nil {
		return nil, err
	}

	return p.Stream, nil
}

// EndData marks the end of the data transfer. It closes the pipeline's
// TransferStream, and moves the file (when needed) from it's temporary location
// to its final destination.
func (p *Pipeline) EndData() *types.TransferError {
	if err := p.machine.Transition(stateDataEnd); err != nil {
		p.handleStateErr("EndData", p.machine.Current())

		return errStateMachine
	}

	Tester.dataDone(p.TransCtx.Transfer)

	if err := p.Stream.close(); err != nil {
		return err
	}

	if err := p.Stream.move(); err != nil {
		return err
	}

	if p.TransCtx.Transfer.Filesize < 0 {
		if err := p.checkFileExist(); err != nil {
			p.handleError(err.Code, "Error during final file check", err.Details)

			return err
		}
	}

	if err := p.machine.Transition(stateDataEndDone); err != nil {
		p.handleStateErr("EndDataDone", p.machine.Current())

		return errStateMachine
	}

	return nil
}

// PostTasks executes the transfer's post-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
//
//nolint:dupl // factorizing would add complexity
func (p *Pipeline) PostTasks() *types.TransferError {
	defer Tester.postTasksDone(p.TransCtx.Transfer)

	if err := p.machine.Transition(statePostTasks); err != nil {
		p.handleStateErr("PostTasks", p.machine.Current())

		return errStateMachine
	}

	if p.TransCtx.Transfer.Step > types.StepPostTasks {
		if err := p.machine.Transition(statePostTasksDone); err != nil {
			p.handleStateErr("PostTasksDone", p.machine.Current())

			return errStateMachine
		}

		return nil
	}

	p.TransCtx.Transfer.Step = types.StepPostTasks
	if dbErr := p.UpdateTrans(); dbErr != nil {
		return dbErr
	}

	if err := p.runner.PostTasks(); err != nil {
		p.handleError(err.Code, "Post-tasks failed", err.Details)

		return types.NewTransferError(types.TeExternalOperation, "post-tasks failed")
	}

	if err := p.machine.Transition(statePostTasksDone); err != nil {
		p.handleStateErr("PostTasksDone", p.machine.Current())

		return errStateMachine
	}

	return nil
}

// EndTransfer signals that the transfer has ended normally, and archives in the
// transfer history.
func (p *Pipeline) EndTransfer() *types.TransferError {
	if err := p.machine.Transition(stateEndTransfer); err != nil {
		p.handleStateErr("EndTransfer", p.machine.Current())

		return errStateMachine
	}

	var sErr *types.TransferError

	p.errOnce.Do(func() {
		p.runner.Stop()
		p.TransCtx.Transfer.Status = types.StatusDone
		p.TransCtx.Transfer.Step = types.StepNone
		p.TransCtx.Transfer.TaskNumber = 0

		if err := p.TransCtx.Transfer.ToHistory(p.DB, p.Logger, time.Now()); err != nil {
			if mErr := p.machine.Transition(stateError); mErr != nil {
				p.Logger.Warning("Failed to transition to 'error' state: %v", mErr)
			}

			p.errDo(types.TeInternal, "Failed to archive transfer", err.Error())
			sErr = errDatabase

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
		Tester.errTasksDone(p.TransCtx.Transfer)
		p.TransCtx.Transfer.Step = oldStep
		p.TransCtx.Transfer.TaskNumber = oldTask

		if dbErr := p.UpdateTrans(); dbErr != nil {
			p.Logger.Error("Failed to reset transfer step after error-tasks: %s", dbErr)
		}
	}()

	p.TransCtx.Transfer.TaskNumber = 0
	p.TransCtx.Transfer.Step = types.StepErrorTasks

	if dbErr := p.UpdateTrans(); dbErr != nil {
		p.Logger.Error("Failed to update transfer step for error-tasks: %s", dbErr)
	}

	if err := p.runner.ErrorTasks(); err != nil {
		p.Logger.Error("Error-tasks failed: %s", err.Details)
	}
}

func (p *Pipeline) errDo(code types.TransferErrorCode, msg, cause string) {
	fullMsg := fmt.Sprintf("%s: %s", msg, cause)

	p.Logger.Error(fullMsg)
	p.runner.Stop()

	if p.Stream != nil {
		p.Stream.stop()
	}

	go func() {
		p.TransCtx.Transfer.Error = *types.NewTransferError(code, fullMsg)
		if dbErr := p.UpdateTrans(); dbErr != nil {
			p.Logger.Error("Failed to update transfer error: %s", dbErr)
		}

		p.errorTasks()
		p.doneErr(types.StatusError)
	}()
}

func (p *Pipeline) handleError(code types.TransferErrorCode, msg, cause string) {
	p.errOnce.Do(func() {
		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warning("Failed to transition to 'error' state: %v", mErr)
		}

		p.errDo(code, msg, cause)
	})
}

func (p *Pipeline) handleStateErr(fun string, currentState statemachine.State) {
	p.handleError(types.TeInternal, "Pipeline state machine violation", fmt.Sprintf(
		"cannot call %s while in state %s", fun, currentState))
}

// SetError stops the pipeline and sets its error to the given value.
func (p *Pipeline) SetError(err *types.TransferError) {
	p.errOnce.Do(func() {
		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warning("Failed to transition to 'error' state: %v", mErr)
		}

		p.stop()
		p.errDo(err.Code, "Error on remote partner", err.Details)
	})
}

func (p *Pipeline) stop() {
	switch p.machine.Last() {
	case statePreTasks, statePostTasks:
		p.runner.Stop()
	case stateReading, stateWriting, stateDataEnd:
		p.Stream.stop()
	}
}

// Pause stops the pipeline and pauses the transfer.
func (p *Pipeline) Pause(handles ...func()) {
	p.halt("Transfer paused by user", types.StatusPaused, handles...)
}

// Interrupt stops the pipeline and interrupts the transfer.
func (p *Pipeline) Interrupt(handles ...func()) {
	p.halt("Transfer interrupted by a service shutdown", types.StatusInterrupted, handles...)
}

func (p *Pipeline) halt(msg string, status types.TransferStatus, handles ...func()) {
	p.errOnce.Do(func() {
		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warning("Failed to transition to 'error' state: %v", mErr)
		}

		for _, handle := range handles {
			handle()
		}

		p.Logger.Info(msg)
		p.stop()
		p.doneErr(status)
	})
}

// Cancel stops the pipeline and cancels the transfer.
func (p *Pipeline) Cancel(handles ...func()) {
	p.errOnce.Do(func() {
		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warning("Failed to transition to 'error' state: %v", mErr)
		}

		for _, handle := range handles {
			handle()
		}

		p.Logger.Info("Transfer canceled by user")
		p.stop()

		p.TransCtx.Transfer.Status = types.StatusCancelled
		if err := p.TransCtx.Transfer.ToHistory(p.DB, p.Logger, time.Now()); err != nil {
			p.Logger.Error("Failed to move canceled transfer to history: %s", err)
		}

		p.done(stateInError)
	})
}

// RebuildFilepaths rebuilds the transfer's local and remote paths, just like it
// is done at the beginning of the transfer. This is useful if the file's name
// has changed during the transfer.
func (p *Pipeline) RebuildFilepaths() *types.TransferError {
	p.setFilePaths()

	if err := p.UpdateTrans(); err != nil {
		p.handleError(types.TeInternal, "Failed to update the transfer file paths",
			err.Error())

		return errDatabase
	}

	return nil
}

func (p *Pipeline) doneErr(status types.TransferStatus) {
	p.TransCtx.Transfer.Status = status
	if err := p.DB.Update(p.TransCtx.Transfer).Run(); err != nil {
		p.Logger.Error("Failed to update transfer status to %v: %s", status, err)
	}

	p.done(stateInError)
}

func (p *Pipeline) doneOK() {
	defer p.Logger.Debug("Transfer ended without errors")

	p.done(stateAllDone)
}

func (p *Pipeline) done(state statemachine.State) {
	if mErr := p.machine.Transition(state); mErr != nil {
		p.Logger.Warning("Failed to transition to '%s' state: %v", state, mErr)
	}

	if !p.TransCtx.Transfer.IsServer {
		TransferOutCount.Sub()
	} else {
		TransferInCount.Sub()
	}

	Tester.done(p.TransCtx.Transfer.IsServer)
}
