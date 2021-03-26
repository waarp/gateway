package pipeline

import (
	"fmt"
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

// Pipeline is the structure regrouping all elements of the transfer Pipeline
// which are not protocol-dependent, such as task execution.
type Pipeline struct {
	db       *database.DB
	logger   *log.Logger
	transCtx *model.TransferContext

	machine *internal.Machine
	errOnce sync.Once

	stream TransferStream
	runner *tasks.Runner
}

func newPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext) (*Pipeline, error) {
	runner := tasks.NewTaskRunner(db, logger, transCtx)
	internal.MakeFilepaths(transCtx)

	if transCtx.Rule.IsSend {
		if err := internal.CheckFileExist(transCtx.Transfer, logger); err != nil {
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
		db:       db,
		logger:   logger,
		transCtx: transCtx,
		machine:  internal.PipelineSateMachine.New(),
		stream:   nil,
		runner:   runner,
	}, nil
}

func (p *Pipeline) PreTasks() error {
	if err := p.machine.Transition("pre-tasks"); err != nil {
		p.handleStateErr("PreTasks", p.machine.Current())
		return errStateMachine
	}

	if p.transCtx.Transfer.Step > types.StepPreTasks {
		if err := p.machine.Transition("pre-tasks done"); err != nil {
			p.handleStateErr("PreTasksDone", p.machine.Current())
			return errStateMachine
		}
		return nil
	}
	p.transCtx.Transfer.Step = types.StepPreTasks
	if dbErr := p.db.Update(p.transCtx.Transfer).Cols("step").Run(); dbErr != nil {
		p.handleError(types.TeInternal, "Failed to update transfer step for pre-tasks",
			dbErr.Error())
		return errDatabase
	}

	if err := p.runner.PreTasks(); err != nil {
		if pErr, ok := err.(types.TransferError); ok {
			p.handleError(pErr.Code, "Pre-tasks failed", pErr.Details)
			return types.NewTransferError(types.TeExternalOperation, "pre-tasks failed")
		}
		p.handleError(types.TeExternalOperation, "Pre-tasks failed", err.Error())
		return types.NewTransferError(types.TeExternalOperation, "pre-tasks failed")
	}

	if err := p.machine.Transition("pre-tasks done"); err != nil {
		p.handleStateErr("PreTasksDone", p.machine.Current())
		return errStateMachine
	}

	return nil
}

func (p *Pipeline) StartData() (TransferStream, error) {
	if err := p.machine.Transition("start data"); err != nil {
		p.handleStateErr("StartData", p.machine.Current())
		return nil, errStateMachine
	}

	if p.transCtx.Transfer.Step > types.StepPreTasks {
		var err error
		if p.stream, err = newVoidStream(p); err != nil {
			p.handleError(types.TeInternal, "Failed to create file stream", err.Error())
			return nil, err
		}
		return p.stream, nil
	}
	p.transCtx.Transfer.Step = types.StepPreTasks
	if dbErr := p.db.Update(p.transCtx.Transfer).Cols("step").Run(); dbErr != nil {
		p.handleError(types.TeInternal, "Failed to update transfer step for pre-tasks",
			dbErr.Error())
		return nil, errDatabase
	}

	var err error
	p.stream, err = newFileStream(p, time.Second)
	if err != nil {
		if tErr, ok := err.(types.TransferError); ok {
			p.handleError(tErr.Code, "Failed to create file stream", tErr.Details)
			return nil, tErr
		}
		p.handleError(types.TeInternal, "Failed to create file stream", err.Error())
		return nil, err
	}

	return p.stream, nil
}

func (p *Pipeline) EndData() error {
	if err := p.stream.close(); err != nil {
		return err
	}
	if err := p.stream.move(); err != nil {
		return err
	}

	if err := p.machine.Transition("end data"); err != nil {
		p.handleStateErr("EndDataDone", p.machine.Current())
		return errStateMachine
	}
	return nil
}

func (p *Pipeline) PostTasks() error {
	if err := p.machine.Transition("post-tasks"); err != nil {
		p.handleStateErr("PostTasks", p.machine.Current())
		return errStateMachine
	}

	if p.transCtx.Transfer.Step > types.StepPostTasks {
		if err := p.machine.Transition("post-tasks done"); err != nil {
			p.handleStateErr("PostTasksDone", p.machine.Current())
			return errStateMachine
		}
		return nil
	}

	p.transCtx.Transfer.Step = types.StepPostTasks
	if dbErr := p.db.Update(p.transCtx.Transfer).Cols("step").Run(); dbErr != nil {
		p.handleError(types.TeInternal, "Failed to update transfer step for post-tasks",
			dbErr.Error())
		return errDatabase
	}

	if err := p.runner.PostTasks(); err != nil {
		if pErr, ok := err.(types.TransferError); ok {
			p.handleError(pErr.Code, "Post-tasks failed", pErr.Details)
			return types.NewTransferError(types.TeExternalOperation, "post-tasks failed")
		}
		p.handleError(types.TeExternalOperation, "Post-tasks failed", err.Error())
		return types.NewTransferError(types.TeExternalOperation, "post-tasks failed")
	}

	if err := p.machine.Transition("post-tasks done"); err != nil {
		p.handleStateErr("PostTasksDone", p.machine.Current())
		return errStateMachine
	}

	return nil
}

func (p *Pipeline) EndTransfer() error {
	if err := p.machine.Transition("end transfer"); err != nil {
		p.handleStateErr("EndTransfer", p.machine.Current())
		return errStateMachine
	}
	p.runner.Stop()
	p.transCtx.Transfer.Status = types.StatusDone
	p.transCtx.Transfer.Step = types.StepNone
	if err := p.transCtx.Transfer.ToHistory(p.db, p.logger); err != nil {
		p.handleError(types.TeInternal, "Failed to archive transfer", err.Error())
		return errDatabase
	}
	if err := p.machine.Transition("all done"); err != nil {
		p.handleStateErr("Done", p.machine.Current())
		return errStateMachine
	}

	p.logger.Debug("Transfer ended without errors")
	return nil
}

func (p *Pipeline) errorTasks() {
	oldStep := p.transCtx.Transfer.Step
	oldTask := p.transCtx.Transfer.TaskNumber
	defer func() {
		p.transCtx.Transfer.Step = oldStep
		p.transCtx.Transfer.TaskNumber = oldTask
		if dbErr := p.db.Update(p.transCtx.Transfer).Cols("step", "task_number").Run(); dbErr != nil {
			p.logger.Errorf("Failed to reset transfer step after error-tasks: %s", dbErr)
		}
	}()

	p.transCtx.Transfer.TaskNumber = 0
	p.transCtx.Transfer.Step = types.StepPostTasks
	if dbErr := p.db.Update(p.transCtx.Transfer).Cols("step", "task_number").Run(); dbErr != nil {
		p.logger.Errorf("Failed to update transfer step for error-tasks: %s", dbErr)
	}

	if err := p.runner.ErrorTasks(); err != nil {
		if tErr, ok := err.(types.TransferError); ok {
			p.logger.Errorf("Error-tasks failed: %s", tErr.Details)
		} else {
			p.logger.Errorf("Error-tasks failed: %s", err)
		}
	}
}

func (p *Pipeline) errDo(code types.TransferErrorCode, msg, cause string) {
	_ = p.machine.Transition("error")
	fullMsg := fmt.Sprintf("%s: %s", msg, cause)
	p.logger.Error(fullMsg)

	p.runner.Stop()

	go func() {
		p.transCtx.Transfer.Error = types.NewTransferError(code, fmt.Sprintf("%s: %s", msg, cause))
		if dbErr := p.db.Update(p.transCtx.Transfer).Cols("progress", "error_code",
			"error_details").Run(); dbErr != nil {
			p.logger.Errorf("Failed to update transfer error: %s", dbErr)
		}

		p.errorTasks()

		p.transCtx.Transfer.Status = types.StatusError
		if dbErr := p.db.Update(p.transCtx.Transfer).Cols("status").Run(); dbErr != nil {
			p.logger.Errorf("Failed to update transfer status to ERROR: %s", dbErr)
		}

		if err := p.machine.Transition("in error"); err != nil {
			p.handleStateErr("ErrorDone", p.machine.Current())
			return
		}
	}()
}

func (p *Pipeline) handleError(code types.TransferErrorCode, msg, cause string) {
	p.errOnce.Do(func() { p.errDo(code, msg, cause) })
}

func (p *Pipeline) handleStateErr(fun, currentState string) {
	p.handleError(types.TeInternal, "Pipeline state machine violation", fmt.Sprintf(
		"cannot call %s while in state %s", fun, currentState))
}

func (p *Pipeline) SetError(err error) {
	p.errOnce.Do(func() {
		code := types.TeUnknownRemote
		details := err.Error()
		if tErr, ok := err.(types.TransferError); ok {
			code = tErr.Code
			details = tErr.Details
		}

		p.stop()

		p.errDo(code, "An error occurred on the remote partner", details)
	})
}

func (p *Pipeline) stop() {
	switch p.machine.Current() {
	case "error":
		return
	case "pre-tasks", "post-tasks":
		p.runner.Stop()
	case "reading", "writing", "close":
		p.stream.stop()
	}
}

func (p *Pipeline) Pause() {
	p.errOnce.Do(func() {
		p.logger.Info("Transfer paused by user")
		p.stop()
		_ = p.machine.Transition("error")

		p.transCtx.Transfer.Status = types.StatusPaused
		if err := p.db.Update(p.transCtx.Transfer).Cols("status").Run(); err != nil {
			p.logger.Errorf("Failed to update transfer status to PAUSED: %s", err)
		}

		_ = p.machine.Transition("in error")
	})
}

func (p *Pipeline) interrupt() {
	p.errOnce.Do(func() {
		p.logger.Info("Transfer interrupted by a service shutdown")
		p.stop()
		_ = p.machine.Transition("error")

		p.transCtx.Transfer.Status = types.StatusInterrupted
		if err := p.db.Update(p.transCtx.Transfer).Cols("status").Run(); err != nil {
			p.logger.Errorf("Failed to update transfer status to INTERRUPTED: %s", err)
		}

		_ = p.machine.Transition("in error")
	})
}

func (p *Pipeline) Cancel() {
	p.errOnce.Do(func() {
		p.logger.Info("Transfer cancelled by user")
		p.stop()
		_ = p.machine.Transition("error")

		p.transCtx.Transfer.Status = types.StatusCancelled
		if err := p.transCtx.Transfer.ToHistory(p.db, p.logger); err != nil {
			p.logger.Errorf("Failed to move cancelled transfer to history: %s", err)
		}

		_ = p.machine.Transition("in error")
	})
}
