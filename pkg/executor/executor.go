// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"fmt"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

// ClientConstructor is the type representing the constructors used to make new
// instances of transfer clients. All transfer clients must have a ClientConstructor
// function in order to be called by the transfer executor.
type ClientConstructor func(model.OutTransferInfo, <-chan model.Signal) (pipeline.Client, error)

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
}

func (e *Executor) getClient(stream *pipeline.TransferStream) (client pipeline.Client,
	te model.TransferError) {

	info, err := model.NewOutTransferInfo(e.Db, stream.Transfer)
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve transfer info: %s", err)
		e.Logger.Critical(msg)
		te = model.NewTransferError(model.TeInternal, msg)
		return
	}

	constr, ok := ClientsConstructors[info.Agent.Protocol]
	if !ok {
		msg := fmt.Sprintf("Unknown transfer protocol")
		e.Logger.Critical(msg)
		te = model.NewTransferError(model.TeConnection, msg)
		return
	}
	client, err = constr(*info, stream.Signals)
	if err != nil {
		msg := fmt.Sprintf("Failed to create transfer client: %s", err)
		e.Logger.Critical(msg)
		te = model.NewTransferError(model.TeConnection, msg)
		return
	}

	return client, te
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

func (e *Executor) data(stream *pipeline.TransferStream, client pipeline.Client) model.TransferError {
	stream.Transfer.Status = model.StatusTransfer
	if err := stream.Transfer.Update(e.Db); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	if err := client.Data(stream); err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return model.NewTransferError(model.TeDataTransfer, err.Error())
	}
	return model.NewTransferError(model.TeOk, "")
}

func (e *Executor) runTransfer(stream *pipeline.TransferStream) {
	client, cErr := e.getClient(stream)
	if cErr.Code != model.TeOk {
		stream.Transfer.Error = cErr
		stream.Exit()
		return
	}

	if pErr := e.prologue(client); pErr.Code != model.TeOk {
		stream.Transfer.Error = pErr
		stream.Exit()
		return
	}

	tErr := func() model.TransferError {
		if pErr := stream.PreTasks(); pErr.Code != model.TeOk {
			return pErr
		}
		if dErr := e.data(stream, client); dErr.Code != model.TeOk {
			return dErr
		}
		if pErr := stream.PostTasks(); pErr.Code != model.TeOk {
			return pErr
		}

		stream.Exit()
		return model.TransferError{}
	}()

	if tErr.Code != model.TeOk {
		if tErr.Code == model.TeShuttingDown {
			stream.Transfer.Error = tErr
			stream.Exit()
			return
		}

		stream.ErrorTasks(tErr)
		stream.Exit()
	}
}

// Run starts the transfer executor. The executor will execute transfers received
// on the transfers channel until the `Close` method is called.
func (e *Executor) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		for trans := range e.Transfers {
			stream, err := pipeline.NewTransferStream(e.Logger, e.Db, "", trans)
			if err != nil {
				e.Logger.Errorf("Failed to create transfer stream: %s", err.Error())
			}
			e.runTransfer(stream)
		}
		wg.Done()
	}()
}
