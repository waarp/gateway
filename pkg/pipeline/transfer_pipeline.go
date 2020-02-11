package pipeline

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// Pipeline is the structure regrouping all elements of the transfer pipeline
// which are not protocol-dependent, such as task execution.
type Pipeline struct {
	Db       *database.Db
	Logger   *log.Logger
	Root     string
	Transfer *model.Transfer

	Rule    *model.Rule
	Signals chan model.Signal
	proc    *tasks.Processor
}

// PreTasks executes the transfer's pre-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PreTasks() *model.PipelineError {
	return execTasks(p.proc, model.ChainPre, model.StepPreTasks)
}

// PostTasks executes the transfer's post-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PostTasks() *model.PipelineError {
	return execTasks(p.proc, model.ChainPost, model.StepPostTasks)
}

// ErrorTasks updates the transfer's error in the database with the given one,
// and then executes the transfer's error-tasks.
func (p *Pipeline) ErrorTasks() {
	failedStep := p.Transfer.Step
	_ = execTasks(p.proc, model.ChainError, model.StepErrorTasks)
	p.Transfer.Step = failedStep
}

// Archive deletes the transfer entry and saves it in the history.
func (p *Pipeline) Archive() {
	toHistory(p.Db, p.Logger, p.Transfer)
}

// Exit deletes the transfer's signal channel.
func (p *Pipeline) Exit() {
	Signals.Delete(p.Transfer.ID)
}
