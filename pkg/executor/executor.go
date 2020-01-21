// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"fmt"
	"os"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// ClientConstructor is the type representing the constructors used to make new
// instances of transfer clients. All transfer clients must have a ClientConstructor
// function in order to be called by the transfer executor.
type ClientConstructor func(model.OutTransferInfo, *os.File, <-chan model.Signal) (pipeline.Client, error)

var (
	// ClientsConstructors is a map associating a protocol to its client constructor.
	ClientsConstructors = map[string]ClientConstructor{}
)

// Executor is the process responsible for executing outgoing transfers.
type Executor struct {
	Db        *database.Db
	Logger    *log.Logger
	R66Home   string
	Transfers <-chan model.Transfer

	finished chan bool
}

func (e *Executor) setup(trans model.Transfer, s <-chan model.Signal) (client pipeline.Client,
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
func (e *Executor) handleError(id uint64, err model.TransferError) {
	if dbErr := e.Db.Update(&model.Transfer{Error: err}, id, false); dbErr != nil {
		e.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
		return
	}
	pipeline.ToHistory(e.Db, e.Logger, id, true)
}

func (e *Executor) runTransfer(trans model.Transfer, s <-chan model.Signal) {
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
		if pErr := pipeline.PreTasks(e.Db, e.Logger, proc); pErr.Code != model.TeOk {
			return pErr
		}
		if dErr := e.data(trans.ID, client); dErr.Code != model.TeOk {
			return dErr
		}
		if pErr := pipeline.PostTasks(e.Db, e.Logger, proc); pErr.Code != model.TeOk {
			return pErr
		}
		pipeline.ToHistory(e.Db, e.Logger, trans.ID, false)
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
		if eErr := pipeline.ErrorTasks(e.Db, e.Logger, proc, tErr); eErr.Code != model.TeOk {
			return
		}
		pipeline.ToHistory(e.Db, e.Logger, trans.ID, true)
	}
}

// Run starts the transfer executor. The executor will execute transfers received
// on the transfers channel until the `Close` method is called.
func (e *Executor) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		e.finished = make(chan bool)
		for trans := range e.Transfers {
			s := make(chan model.Signal)
			pipeline.Signals.Store(trans.ID, s)
			e.runTransfer(trans, s)
			pipeline.Signals.Delete(trans.ID)
		}
		close(e.finished)
		wg.Done()
	}()
}
