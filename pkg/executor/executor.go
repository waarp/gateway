// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"fmt"
	"os"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// ClientConstructor is the type representing the constructors used to make new
// instances of transfer clients. All transfer clients must have a ClientConstructor
// function in order to be called by the transfer executor.
type ClientConstructor func(model.OutTransferInfo, *os.File, <-chan pipeline.Signal) (pipeline.Client, error)

var (
	// ClientsConstructors is a map associating a protocol to its client constructor.
	ClientsConstructors = map[string]ClientConstructor{}

	// Signals is a map regrouping the signal channels of all ongoing transfers.
	// The signal channel of a specific transfer can be retrieved from this map
	// using the transfer's ID.
	Signals sync.Map
)

// Executor is the process responsible for executing outgoing transfers.
type Executor struct {
	Db        *database.Db
	Logger    *log.Logger
	R66Home   string
	Transfers <-chan model.Transfer

	finished chan bool
}

func (e *Executor) toHistory(id uint64, isError bool) {
	trans := &model.Transfer{ID: id}

	if err := e.Db.Get(trans); err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer for archival: %s", err)
		return
	}

	ses, err := e.Db.BeginTransaction()
	if err != nil {
		e.Logger.Criticalf("Failed to start archival transaction: %s", err)
		return
	}

	if err := ses.Delete(&model.Transfer{ID: id}); err != nil {
		e.Logger.Criticalf("Failed to delete transfer for archival: %s", err)
		ses.Rollback()
		return
	}

	if isError {
		trans.Status = model.StatusError
	} else {
		trans.Status = model.StatusDone
	}

	hist, err := trans.ToHistory(ses, time.Now())
	if err != nil {
		e.Logger.Criticalf("Failed to convert transfer to history: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Create(hist); err != nil {
		e.Logger.Criticalf("Failed to create new history entry: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Commit(); err != nil {
		e.Logger.Criticalf("Failed to commit archival transaction: %s", err)
		return
	}
}

func (e *Executor) setup(trans model.Transfer, s <-chan pipeline.Signal) (client pipeline.Client,
	proc *tasks.Processor, te model.TransferError) {

	info, err := model.NewOutTransferInfo(e.Db, trans)
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve transfer info: %s", err)
		e.Logger.Critical(msg)
		te = model.NewTransferError(model.TeInternal, msg)
		return
	}

	var file *os.File
	if info.Rule.IsSend {
		file, err = os.OpenFile(info.Transfer.SourcePath, os.O_RDONLY, 0600)
		if err != nil {
			msg := fmt.Sprintf("Failed to open local file: %s", err)
			e.Logger.Critical(msg)
			te = model.NewTransferError(model.TeFileNotFound, msg)
			return
		}
	} else {
		file, err = os.OpenFile(info.Transfer.SourcePath, os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			msg := fmt.Sprintf("Failed to create file: %s", err)
			e.Logger.Critical(msg)
			te = model.NewTransferError(model.TeForbidden, msg)
			return
		}
	}

	constr, ok := ClientsConstructors[info.Agent.Protocol]
	if !ok {
		msg := fmt.Sprintf("Unknown transfer protocol")
		e.Logger.Critical(msg)
		te = model.NewTransferError(model.TeConnection, msg)
		return
	}
	client, err = constr(*info, file, s)
	if err != nil {
		msg := fmt.Sprintf("Failed to create transfer client: %s", err)
		e.Logger.Critical(msg)
		te = model.NewTransferError(model.TeConnection, msg)
		return
	}

	proc = &tasks.Processor{
		Db:       e.Db,
		Logger:   e.Logger,
		Rule:     &info.Rule,
		Transfer: &info.Transfer,
		Signals:  s,
	}

	return client, proc, te
}

func (e *Executor) prologue(client pipeline.Client) model.TransferError {
	if err := client.Connect(); err != nil {
		msg := fmt.Sprintf("Failed to connect to remote agent: %s", err)
		e.Logger.Error(msg)
		return model.NewTransferError(model.TeConnection, msg)
	}

	if err := client.Authenticate(); err != nil {
		msg := fmt.Sprintf("Failed to authenticate on remote agent: %s", err)
		e.Logger.Error(msg)
		return model.NewTransferError(model.TeBadAuthentication, msg)
	}

	if err := client.Request(); err != nil {
		msg := fmt.Sprintf("Failed to make transfer request: %s", err)
		e.Logger.Error(msg)
		return model.NewTransferError(model.TeUnknown, msg)
	}

	return model.NewTransferError(model.TeOk, "")
}

func (e *Executor) preTasks(id uint64, proc *tasks.Processor) model.TransferError {
	stat := &model.Transfer{Status: model.StatusPreTasks}
	if err := e.Db.Update(stat, id, false); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	preTasks, err := proc.GetTasks(model.ChainPre)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve pre-tasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	return proc.RunTasks(preTasks)
}

func (e *Executor) data(id uint64, client pipeline.Client) model.TransferError {
	stat := &model.Transfer{Status: model.StatusTransfer}
	if err := e.Db.Update(stat, id, false); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	if err := client.Data(); err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return model.NewTransferError(model.TeDataTransfer, err.Error())
	}
	return model.NewTransferError(model.TeOk, "")
}

func (e *Executor) postTasks(id uint64, proc *tasks.Processor) model.TransferError {
	stat := &model.Transfer{Status: model.StatusPostTasks}
	if err := e.Db.Update(stat, id, false); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	postTasks, err := proc.GetTasks(model.ChainPost)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve post-tasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	return proc.RunTasks(postTasks)
}

func (e *Executor) errorTasks(id uint64, proc *tasks.Processor) model.TransferError {
	stat := &model.Transfer{Status: model.StatusErrorTasks}
	if err := e.Db.Update(stat, id, false); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	errorTasks, err := proc.GetTasks(model.ChainError)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve error-tasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	return proc.RunTasks(errorTasks)
}

func (e *Executor) handleError(id uint64, err model.TransferError) {
	if dbErr := e.Db.Update(&model.Transfer{Error: err}, id, false); dbErr != nil {
		e.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
		return
	}
	e.toHistory(id, true)
}

func (e *Executor) runTransfer(trans model.Transfer, s <-chan pipeline.Signal) {
	client, proc, sErr := e.setup(trans, s)
	if sErr.Code != model.TeOk {
		e.handleError(trans.ID, sErr)
		return
	}

	if pErr := e.prologue(client); pErr.Code != model.TeOk {
		e.handleError(trans.ID, pErr)
		return
	}

	tErr := func() model.TransferError {
		if pErr := e.preTasks(trans.ID, proc); pErr.Code != model.TeOk {
			return pErr
		}
		if dErr := e.data(trans.ID, client); dErr.Code != model.TeOk {
			return dErr
		}
		if pErr := e.postTasks(trans.ID, proc); pErr.Code != model.TeOk {
			return pErr
		}
		e.toHistory(trans.ID, false)
		return model.NewTransferError(model.TeOk, "")
	}()

	if tErr.Code == model.TeShuttingDown {
		e.handleError(trans.ID, tErr)
		return
	}
	if tErr.Code != model.TeOk {
		if dbErr := e.Db.Update(&model.Transfer{Error: tErr}, trans.ID, false); dbErr != nil {
			e.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
		if eErr := e.errorTasks(trans.ID, proc); eErr.Code != model.TeOk {
			return
		}
		e.toHistory(trans.ID, true)
	}
}

// Run starts the transfer executor. The executor will execute transfers received
// on the transfers channel until the `Close` method is called.
func (e *Executor) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		e.finished = make(chan bool)
		for trans := range e.Transfers {
			s := make(chan pipeline.Signal)
			Signals.Store(trans.ID, s)
			e.runTransfer(trans, s)
			Signals.Delete(trans.ID)
		}
		close(e.finished)
		wg.Done()
	}()
}
