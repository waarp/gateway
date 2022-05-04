package pipeline

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
)

var (
	errDatabase     = types.NewTransferError(types.TeInternal, "database error")
	errStateMachine = types.NewTransferError(types.TeInternal, "internal transfer error")
)

// Pipeline is the structure regrouping all elements of the transfer Pipeline
// which are not protocol-dependent, such as task execution.
//
// A typical transfer's execution order should be as follow:
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

	machine *statemachine.Machine
	errOnce sync.Once

	runner *tasks.Runner
}

func NewPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) (*Pipeline, []string, *types.TransferError) {
	cols := []string{"progression", "filesize"}

	pipeline := &Pipeline{
		DB:       db,
		Logger:   logger,
		TransCtx: transCtx,
		machine:  pipelineSateMachine.New(),
		runner:   tasks.NewTaskRunner(db, logger, transCtx),
	}

	pipeline.setFilePaths()

	cols = append(cols, "local_path", "remote_path")

	if transCtx.Rule.IsSend {
		if err := pipeline.checkFileExist(); err != nil {
			return nil, cols, err
		}

		cols = append(cols, "filesize")
	}

	if transCtx.Transfer.Status != types.StatusRunning {
		transCtx.Transfer.Status = types.StatusRunning

		cols = append(cols, "status")
	}

	if transCtx.Transfer.Step < types.StepSetup {
		transCtx.Transfer.Step = types.StepSetup

		cols = append(cols, "step")
	}

	if transCtx.Transfer.Error.Code != types.TeOk {
		transCtx.Transfer.Error.Code = types.TeOk

		cols = append(cols, "error_code")
	}

	if transCtx.Transfer.Error.Details != "" {
		transCtx.Transfer.Error.Details = ""

		cols = append(cols, "error_details")
	}

	return pipeline, cols, nil
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
		p.Logger.Errorf("Failed to initiate transfer pipeline: ¨%s", err)
		p.TransCtx.Transfer.Error = *err

		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("error_code", "error_details").
			Run(); dbErr != nil {
			p.Logger.Errorf("Failed to update the transfer error: %s", dbErr)

			return errDatabase
		}
	}

	return err
}

// UpdateTrans updates the given columns of the pipeline's transfer.
func (p *Pipeline) UpdateTrans(cols ...string) *types.TransferError {
	if err := p.DB.Update(p.TransCtx.Transfer).Cols(cols...).Run(); err != nil {
		p.handleError(types.TeInternal, fmt.Sprintf("Failed to update transfer columns %s",
			strings.Join(cols, ", ")), "database error")

		return types.NewTransferError(types.TeInternal, "database error")
	}

	return nil
}

//nolint:dupl // factorizing would add complexity
// PreTasks executes the transfer's pre-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
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
	if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("step").Run(); dbErr != nil {
		p.handleError(types.TeInternal, "Failed to update transfer step for pre-tasks",
			dbErr.Error())

		return errDatabase
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

// SetProgress sets the transfer progress to the given value. Can only be called
// before StartData.
func (p *Pipeline) SetProgress(offset uint64) *types.TransferError {
	if curr := p.machine.Current(); curr != stateInit && curr != statePreTasksDone {
		p.handleStateErr("SetProgress", curr)

		return errStateMachine
	}

	if p.TransCtx.Transfer.Progress == offset {
		return nil
	}

	p.TransCtx.Transfer.Progress = offset

	return p.UpdateTrans("progression")
}

// StartData marks the beginning of the data transfer. It opens the pipeline's
// TransferStream and returns it. In the case of a retry, the data transfer will
// resume at the offset indicated by the Transfer.Progress field.
func (p *Pipeline) StartData() (TransferStream, *types.TransferError) {
	var newState statemachine.State

	if p.TransCtx.Rule.IsSend {
		newState = stateReading
	} else {
		newState = stateWriting
	}

	doTrantition, err := p.machine.DeferTransition(newState)
	if err != nil {
		p.handleStateErr("StartData", p.machine.Current())

		return nil, errStateMachine
	}

	defer doTrantition()

	if p.TransCtx.Transfer.Step > types.StepData {
		p.Stream = newVoidStream(p)

		return p.Stream, nil
	}

	isResume := false
	if p.TransCtx.Transfer.Step == types.StepData {
		isResume = true
	} else {
		p.TransCtx.Transfer.Step = types.StepData
		p.TransCtx.Transfer.TaskNumber = 0

		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("step", "task_number").Run(); dbErr != nil {
			doTrantition()
			p.handleError(types.TeInternal, "Failed to update transfer step for pre-tasks",
				dbErr.Error())

			return nil, errDatabase
		}
	}

	var tErr *types.TransferError

	p.Stream, tErr = newFileStream(p, time.Second, isResume)
	if tErr != nil {
		doTrantition()
		p.handleError(tErr.Code, "Failed to create file stream", tErr.Details)

		return nil, tErr
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

//nolint:dupl // factorizing would add complexity
// PostTasks executes the transfer's post-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
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
	if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("step").Run(); dbErr != nil {
		p.handleError(types.TeInternal, "Failed to update transfer step for post-tasks",
			dbErr.Error())

		return errDatabase
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
				p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
			}

			p.errDo(types.TeInternal, "Failed to archive transfer", err.Error())
			sErr = errDatabase

			return
		}

		if err := p.machine.Transition(stateAllDone); err != nil {
			if mErr := p.machine.Transition(stateError); mErr != nil {
				p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
			}

			p.errDo(types.TeInternal, "Pipeline state machine violation", fmt.Sprintf(
				"cannot call EndTransfer while in state %s", p.machine.Current()))
			sErr = errStateMachine

			return
		}

		p.Logger.Debug("Transfer ended without errors")
		p.done()
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

		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("step", "task_number").Run(); dbErr != nil {
			p.Logger.Errorf("Failed to reset transfer step after error-tasks: %s", dbErr)
		}
	}()

	p.TransCtx.Transfer.TaskNumber = 0
	p.TransCtx.Transfer.Step = types.StepErrorTasks

	if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("step", "task_number").Run(); dbErr != nil {
		p.Logger.Errorf("Failed to update transfer step for error-tasks: %s", dbErr)
	}

	if err := p.runner.ErrorTasks(); err != nil {
		p.Logger.Errorf("Error-tasks failed: %s", err.Details)
	}
}

func (p *Pipeline) errDo(code types.TransferErrorCode, msg, cause string) {
	fullMsg := fmt.Sprintf("%s: %s", msg, cause)

	p.Logger.Error(fullMsg)
	p.runner.Stop()

	go func() {
		p.TransCtx.Transfer.Error = *types.NewTransferError(code, fullMsg)
		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("progress", "error_code",
			"error_details").Run(); dbErr != nil {
			p.Logger.Errorf("Failed to update transfer error: %s", dbErr)
		}

		p.errorTasks()

		p.TransCtx.Transfer.Status = types.StatusError
		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("status").Run(); dbErr != nil {
			p.Logger.Errorf("Failed to update transfer status to ERROR: %s", dbErr)
		}

		if err := p.machine.Transition(stateInError); err != nil {
			p.Logger.Critical(err.Error())

			return
		}

		p.done()
	}()
}

func (p *Pipeline) handleError(code types.TransferErrorCode, msg, cause string) {
	p.errOnce.Do(func() {
		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
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
			p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
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
			p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
		}

		for _, handle := range handles {
			handle()
		}

		p.Logger.Info(msg)
		p.stop()

		p.TransCtx.Transfer.Status = status
		if err := p.DB.Update(p.TransCtx.Transfer).Cols("status").Run(); err != nil {
			p.Logger.Errorf("Failed to update transfer status to %v: %s", status, err)
		}

		if mErr := p.machine.Transition(stateInError); mErr != nil {
			p.Logger.Warningf("Failed to transition to 'in error' state: %v", mErr)
		}

		p.done()
	})
}

// Cancel stops the pipeline and cancels the transfer.
func (p *Pipeline) Cancel(handles ...func()) {
	p.errOnce.Do(func() {
		if mErr := p.machine.Transition(stateError); mErr != nil {
			p.Logger.Warningf("Failed to transition to 'error' state: %v", mErr)
		}

		for _, handle := range handles {
			handle()
		}

		p.Logger.Info("Transfer canceled by user")
		p.stop()

		p.TransCtx.Transfer.Status = types.StatusCancelled
		if err := p.TransCtx.Transfer.ToHistory(p.DB, p.Logger, time.Now()); err != nil {
			p.Logger.Errorf("Failed to move canceled transfer to history: %s", err)
		}

		if mErr := p.machine.Transition(stateInError); mErr != nil {
			p.Logger.Warningf("Failed to transition to 'in error' state: %v", mErr)
		}

		p.done()
	})
}

// RebuildFilepaths rebuilds the transfer's local and remote paths, just like it
// is done at the beginning of the transfer. This is useful if the file's name
// has changed during the transfer.
func (p *Pipeline) RebuildFilepaths() *types.TransferError {
	p.setFilePaths()

	if err := p.DB.Update(p.TransCtx.Transfer).Cols("local_path", "remote_path").Run(); err != nil {
		p.handleError(types.TeInternal, "Failed to update the transfer file paths",
			err.Error())

		return errDatabase
	}

	return nil
}

func (p *Pipeline) done() {
	if !p.TransCtx.Transfer.IsServer {
		TransferOutCount.Sub()
	} else {
		TransferInCount.Sub()
	}

	Tester.done(p.TransCtx.Transfer.IsServer)
}
