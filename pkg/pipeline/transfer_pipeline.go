package pipeline

import (
	"context"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// Pipeline is the structure regrouping all elements of the transfer pipeline
// which are not protocol-dependent, such as task execution.
type Pipeline struct {
	DB       *database.DB
	Logger   *log.Logger
	Transfer *model.Transfer

	Rule    *model.Rule
	Signals chan model.Signal
	Ctx     context.Context
	proc    *tasks.Processor
}

// PreTasks executes the transfer's pre-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PreTasks() *model.PipelineError {
	if p.Transfer.Step > types.StepPreTasks {
		return nil
	}

	p.Logger.Info("Executing pre-tasks")
	return execTasks(p.proc, model.ChainPre, types.StepPreTasks)
}

// PostTasks executes the transfer's post-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PostTasks() *model.PipelineError {
	if p.Transfer.Step > types.StepPostTasks {
		return nil
	}

	p.Logger.Info("Executing post-tasks")
	return execTasks(p.proc, model.ChainPost, types.StepPostTasks)
}

// ErrorTasks updates the transfer's error in the database with the given one,
// and then executes the transfer's error-tasks.
func (p *Pipeline) ErrorTasks() {
	// Save the failed step and task number, and restore then after the error
	// tasks have finished
	failedStep := p.Transfer.Step
	failedTask := p.Transfer.TaskNumber
	defer func() {
		p.Transfer.Step = failedStep
		p.Transfer.TaskNumber = failedTask
	}()
	p.Transfer.TaskNumber = 0

	p.Logger.Info("Executing error tasks")
	_ = execTasks(p.proc, model.ChainError, types.StepErrorTasks)
}

// Archive deletes the transfer entry and saves it in the history.
func (p *Pipeline) Archive() error {
	p.Logger.Info("Transfer finished, saving into transfer history")
	err := ToHistory(p.DB, p.Logger, p.Transfer)
	p.exit()
	return err
}

// exit deletes the transfer's signal channel.
func (p *Pipeline) exit() {
	Signals.Delete(p.Transfer.ID)
	if p.Transfer.IsServer {
		TransferInCount.sub()
	} else {
		TransferOutCount.sub()
	}
}
