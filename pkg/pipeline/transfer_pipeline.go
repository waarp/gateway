package pipeline

import (
	"context"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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
	if p.Transfer.Step == model.StepSetup || p.Transfer.Step == model.StepPreTasks {
		return execTasks(p.proc, model.ChainPre, model.StepPreTasks)
	}
	return nil
}

// PostTasks executes the transfer's post-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PostTasks() *model.PipelineError {
	p.Transfer.Progress = 0
	if err := p.Transfer.Update(p.DB); err != nil {
		p.Logger.Errorf("Failed to update transfer progress: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	if p.Transfer.Step == model.StepData || p.Transfer.Step == model.StepPostTasks {
		return execTasks(p.proc, model.ChainPost, model.StepPostTasks)
	}
	return nil
}

// ErrorTasks updates the transfer's error in the database with the given one,
// and then executes the transfer's error-tasks.
func (p *Pipeline) ErrorTasks() {
	if p.Transfer.Step == model.StepFinalization {
		return
	}
	failedStep := p.Transfer.Step
	failedTask := p.Transfer.TaskNumber
	p.Transfer.TaskNumber = 0
	_ = execTasks(p.proc, model.ChainError, model.StepErrorTasks)
	p.Transfer.Step = failedStep
	p.Transfer.TaskNumber = failedTask
}

// Archive deletes the transfer entry and saves it in the history.
func (p *Pipeline) Archive() {
	_ = ToHistory(p.DB, p.Logger, p.Transfer)
}

// Exit deletes the transfer's signal channel.
func (p *Pipeline) Exit() {
	Signals.Delete(p.Transfer.ID)
	if p.Transfer.IsServer {
		TransferInCount.sub()
	} else {
		TransferOutCount.sub()
	}
}
