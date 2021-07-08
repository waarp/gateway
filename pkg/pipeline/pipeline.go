package pipeline

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/internal"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

var (
	errDatabase     = types.NewTransferError(types.TeInternal, "database error")
	errStateMachine = types.NewTransferError(types.TeInternal, "internal transfer error")
)

// TestPipelineEnd is a function, which (if not nil) is called when a pipeline
// has reached its end. Should only be used for test purposes to signal when a
// transfer has truly ended.
var TestPipelineEnd func(isServer bool)

// Pipeline is the structure regrouping all elements of the transfer Pipeline
// which are not protocol-dependent, such as task execution.
//
// A typical transfer's execution order should be as follow:
// - PreTasks
// - StartData
// - EndData
// - PostTasks
// - EndTransfer
type Pipeline struct {
	DB       *database.DB
	TransCtx *model.TransferContext
	Logger   *log.Logger
	Stream   TransferStream

	machine *internal.Machine
	errOnce sync.Once

	runner *tasks.Runner
}

func newPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) (*Pipeline, *types.TransferError) {
	runner := tasks.NewTaskRunner(db, logger, transCtx)
	internal.MakeFilepaths(transCtx)
	if dbErr := db.Update(transCtx.Transfer).Cols("local_path", "remote_path").Run(); dbErr != nil {
		logger.Errorf("Failed to update transfer paths: %s", dbErr)
		return nil, errDatabase
	}

	if transCtx.Rule.IsSend {
		if err := internal.CheckFileExist(transCtx.Transfer, db, logger); err != nil {
			return nil, err
		}
	}

	if transCtx.Rule.IsSend {
		if !TransferOutCount.Add() {
			return nil, errLimitReached
		}
	} else {
		if !TransferInCount.Add() {
			return nil, errLimitReached
		}
	}

	transCtx.Transfer.Status = types.StatusRunning
	cols := []string{"status"}
	if transCtx.Transfer.Step < types.StepSetup {
		transCtx.Transfer.Step = types.StepSetup
		cols = append(cols, "step")
	} else {
		transCtx.Transfer.Error = types.TransferError{}
		cols = append(cols, "error_code", "error_details")
	}
	if dbErr := db.Update(transCtx.Transfer).Cols(cols...).Run(); dbErr != nil {
		logger.Errorf("Failed to update transfer status to RUNNING: %s", dbErr)
		return nil, errDatabase
	}

	return &Pipeline{
		DB:       db,
		Logger:   logger,
		TransCtx: transCtx,
		machine:  internal.PipelineSateMachine.New(),
		Stream:   nil,
		runner:   runner,
	}, nil
}

// UpdateTrans updates the given columns of the pipeline's transfer.
func (p *Pipeline) UpdateTrans(cols ...string) *types.TransferError {
	if err := p.DB.Update(p.TransCtx.Transfer).Cols(cols...).Run(); err != nil {
		p.handleError(types.TeInternal, fmt.Sprintf("Failed to update transfer %s",
			strings.Join(cols, ", ")), "database error")
		return types.NewTransferError(types.TeInternal, "database error")
	}
	return nil
}

//nolint:dupl
// PreTasks executes the transfer's pre-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
func (p *Pipeline) PreTasks() *types.TransferError {
	if err := p.machine.Transition("pre-tasks"); err != nil {
		p.handleStateErr("PreTasks", p.machine.Current())
		return errStateMachine
	}

	if p.TransCtx.Transfer.Step > types.StepPreTasks {
		if err := p.machine.Transition("pre-tasks done"); err != nil {
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

	if err := p.machine.Transition("pre-tasks done"); err != nil {
		p.handleStateErr("PreTasksDone", p.machine.Current())
		return errStateMachine
	}

	return nil
}

// StartData marks the beginning of the data transfer. It opens the pipeline's
// TransferStream and returns it. In the case of a retry, the data transfer will
// resume at the offset indicated by the Transfer.Progress field.
func (p *Pipeline) StartData() (TransferStream, *types.TransferError) {
	if err := p.machine.Transition("start data"); err != nil {
		p.handleStateErr("StartData", p.machine.Current())
		return nil, errStateMachine
	}

	if p.TransCtx.Transfer.Step > types.StepData {
		var err *types.TransferError
		if p.Stream, err = newVoidStream(p); err != nil {
			p.handleError(types.TeInternal, "Failed to create file stream", err.Error())
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
		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("step", "task_number").Run(); dbErr != nil {
			p.handleError(types.TeInternal, "Failed to update transfer step for pre-tasks",
				dbErr.Error())
			return nil, errDatabase
		}
	}

	var err *types.TransferError
	p.Stream, err = newFileStream(p, time.Second, isResume)
	if err != nil {
		p.handleError(err.Code, "Failed to create file stream", err.Details)
		return nil, err
	}

	return p.Stream, nil
}

// EndData marks the end of the data transfer. It closes the pipeline's
// TransferStream, and moves the file (when needed) from it's temporary location
// to its final destination.
func (p *Pipeline) EndData() *types.TransferError {
	if err := p.machine.Transition("end data"); err != nil {
		p.handleStateErr("EndData", p.machine.Current())
		return errStateMachine
	}
	if err := p.Stream.close(); err != nil {
		return err
	}
	if err := p.Stream.move(); err != nil {
		return err
	}

	if p.TransCtx.Transfer.Filesize < 0 {
		if err := internal.CheckFileExist(p.TransCtx.Transfer, p.DB, p.Logger); err != nil {
			p.handleError(err.Code, "Error during final file check", err.Details)
			return err
		}
	}

	if err := p.machine.Transition("data ended"); err != nil {
		p.handleStateErr("EndDataDone", p.machine.Current())
		return errStateMachine
	}
	return nil
}

//nolint:dupl
// PostTasks executes the transfer's post-tasks. If an error occurs, the pipeline
// is stopped and the transfer's status is set to ERROR.
//
// When resuming a transfer, tasks already successfully executed will be skipped.
func (p *Pipeline) PostTasks() *types.TransferError {
	if err := p.machine.Transition("post-tasks"); err != nil {
		p.handleStateErr("PostTasks", p.machine.Current())
		return errStateMachine
	}

	if p.TransCtx.Transfer.Step > types.StepPostTasks {
		if err := p.machine.Transition("post-tasks done"); err != nil {
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

	if err := p.machine.Transition("post-tasks done"); err != nil {
		p.handleStateErr("PostTasksDone", p.machine.Current())
		return errStateMachine
	}

	return nil
}

// EndTransfer signals that the transfer has ended normally, and archives in the
// transfer history.
func (p *Pipeline) EndTransfer() *types.TransferError {
	if err := p.machine.Transition("end transfer"); err != nil {
		p.handleStateErr("EndTransfer", p.machine.Current())
		return errStateMachine
	}
	p.runner.Stop()
	p.TransCtx.Transfer.Status = types.StatusDone
	p.TransCtx.Transfer.Step = types.StepNone
	p.TransCtx.Transfer.TaskNumber = 0
	if err := p.TransCtx.Transfer.ToHistory(p.DB, p.Logger, time.Now()); err != nil {
		p.handleError(types.TeInternal, "Failed to archive transfer", err.Error())
		return errDatabase
	}
	if err := p.machine.Transition("all done"); err != nil {
		p.handleStateErr("Done", p.machine.Current())
		return errStateMachine
	}

	p.Logger.Debug("Transfer ended without errors")
	if TestPipelineEnd != nil {
		TestPipelineEnd(p.TransCtx.Transfer.IsServer)
	}
	return nil
}

func (p *Pipeline) errorTasks() {
	oldStep := p.TransCtx.Transfer.Step
	oldTask := p.TransCtx.Transfer.TaskNumber
	defer func() {
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
	if err := p.machine.Transition("error"); err != nil {
		p.Logger.Critical(err.Error())
		return
	}
	fullMsg := fmt.Sprintf("%s: %s", msg, cause)
	p.Logger.Error(fullMsg)

	p.runner.Stop()

	go func() {
		p.TransCtx.Transfer.Error = *types.NewTransferError(code, fmt.Sprintf("%s: %s", msg, cause))
		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("progress", "error_code",
			"error_details").Run(); dbErr != nil {
			p.Logger.Errorf("Failed to update transfer error: %s", dbErr)
		}

		p.errorTasks()

		p.TransCtx.Transfer.Status = types.StatusError
		if dbErr := p.DB.Update(p.TransCtx.Transfer).Cols("status").Run(); dbErr != nil {
			p.Logger.Errorf("Failed to update transfer status to ERROR: %s", dbErr)
		}
		if err := p.machine.Transition("in error"); err != nil {
			p.Logger.Critical(err.Error())
			return
		}
		if TestPipelineEnd != nil {
			TestPipelineEnd(p.TransCtx.Transfer.IsServer)
		}
	}()
}

func (p *Pipeline) handleError(code types.TransferErrorCode, msg, cause string) {
	p.errOnce.Do(func() {
		p.errDo(code, msg, cause)
	})
}

func (p *Pipeline) handleStateErr(fun, currentState string) {
	p.handleError(types.TeInternal, "Pipeline state machine violation", fmt.Sprintf(
		"cannot call %s while in state %s", fun, currentState))
}

// SetError stops the pipeline and sets its error to the given value.
func (p *Pipeline) SetError(err *types.TransferError) {
	p.errOnce.Do(func() {
		p.stop()
		p.errDo(err.Code, "Error on remote partner", err.Details)
	})
}

func (p *Pipeline) stop() {
	switch p.machine.Current() {
	case "pre-tasks", "post-tasks":
		p.runner.Stop()
	case "reading", "writing", "close":
		p.Stream.stop()
	}
}

// Pause stops the pipeline and pauses the transfer.
func (p *Pipeline) Pause(handles ...func()) {
	p.errOnce.Do(func() {
		for _, handle := range handles {
			handle()
		}

		p.Logger.Info("Transfer paused by user")
		p.stop()
		_ = p.machine.Transition("error")

		p.TransCtx.Transfer.Status = types.StatusPaused
		if err := p.DB.Update(p.TransCtx.Transfer).Cols("status").Run(); err != nil {
			p.Logger.Errorf("Failed to update transfer status to PAUSED: %s", err)
		}

		_ = p.machine.Transition("in error")
		if TestPipelineEnd != nil {
			TestPipelineEnd(p.TransCtx.Transfer.IsServer)
		}
	})
}

// Interrupt stops the pipeline and interrupts the transfer.
func (p *Pipeline) Interrupt(handles ...func()) {
	p.errOnce.Do(func() {
		for _, handle := range handles {
			handle()
		}

		p.Logger.Info("Transfer interrupted by a service shutdown")
		p.stop()
		_ = p.machine.Transition("error")

		p.TransCtx.Transfer.Status = types.StatusInterrupted
		if err := p.DB.Update(p.TransCtx.Transfer).Cols("status").Run(); err != nil {
			p.Logger.Errorf("Failed to update transfer status to INTERRUPTED: %s", err)
		}

		_ = p.machine.Transition("in error")
		if TestPipelineEnd != nil {
			TestPipelineEnd(p.TransCtx.Transfer.IsServer)
		}
	})
}

// Cancel stops the pipeline and cancels the transfer.
func (p *Pipeline) Cancel(handles ...func()) {
	p.errOnce.Do(func() {
		for _, handle := range handles {
			handle()
		}

		p.Logger.Info("Transfer cancelled by user")
		p.stop()
		_ = p.machine.Transition("error")

		p.TransCtx.Transfer.Status = types.StatusCancelled
		if err := p.TransCtx.Transfer.ToHistory(p.DB, p.Logger, time.Now()); err != nil {
			p.Logger.Errorf("Failed to move cancelled transfer to history: %s", err)
		}

		_ = p.machine.Transition("in error")
		if TestPipelineEnd != nil {
			TestPipelineEnd(p.TransCtx.Transfer.IsServer)
		}
	})
}

// RebuildFilepaths rebuilds the transfer's local and remote paths, just like it
// is done at the beginning of the transfer. This is useful if the file's name
// has changed during the transfer.
func (p *Pipeline) RebuildFilepaths() {
	internal.MakeFilepaths(p.TransCtx)
}
