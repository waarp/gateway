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

	rule    *model.Rule
	Signals chan model.Signal
	proc    *tasks.Processor
}

// PreTasks executes the transfer's pre-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PreTasks() model.TransferError {
	return execTasks(p.proc, model.ChainPre, model.StatusPreTasks)
}

// PostTasks executes the transfer's post-tasks. It returns an error if the
// execution fails.
func (p *Pipeline) PostTasks() model.TransferError {
	return execTasks(p.proc, model.ChainPost, model.StatusPostTasks)
}

// ErrorTasks updates the transfer's error in the database with the given one,
// and then executes the transfer's error-tasks.
func (p *Pipeline) ErrorTasks(te model.TransferError) {
	p.Transfer.Error = te
	if err := p.Transfer.Update(p.Db); err != nil {
		p.Logger.Criticalf("Failed to update transfer error: %s", err.Error())
		return
	}

	_ = execTasks(p.proc, model.ChainError, model.StatusErrorTasks)
}

// Exit deletes the transfer entry and saves it in the history. It also deletes
// the transfer's signal channel.
func (p *Pipeline) Exit() {
	toHistory(p.Db, p.Logger, p.Transfer)
	close(p.Signals)
	Signals.Delete(p.Transfer.ID)
}
