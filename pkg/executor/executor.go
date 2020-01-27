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
	te *model.PipelineError) {

	info, err := model.NewOutTransferInfo(e.Db, stream.Transfer)
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve transfer info: %s", err)
		e.Logger.Critical(msg)
		te = &model.PipelineError{Kind: model.KindDatabase}
		return
	}

	constr, ok := ClientsConstructors[info.Agent.Protocol]
	if !ok {
		msg := fmt.Sprintf("Unknown transfer protocol")
		e.Logger.Critical(msg)
		te = model.NewPipelineError(model.TeConnection, msg)
		return
	}
	client, err = constr(*info, stream.Signals)
	if err != nil {
		msg := fmt.Sprintf("Failed to create transfer client: %s", err)
		e.Logger.Critical(msg)
		te = model.NewPipelineError(model.TeConnection, msg)
		return
	}

	return client, te
}

func (e *Executor) prologue(client pipeline.Client) *model.PipelineError {
	if err := client.Connect(); err != nil {
		msg := fmt.Sprintf("Failed to connect to remote agent: %s", err)
		e.Logger.Error(msg)
		return err
	}

	if err := client.Authenticate(); err != nil {
		msg := fmt.Sprintf("Failed to authenticate on remote agent: %s", err)
		e.Logger.Error(msg)
		return err
	}

	if err := client.Request(); err != nil {
		msg := fmt.Sprintf("Failed to make transfer request: %s", err)
		e.Logger.Error(msg)
		return err
	}

	return nil
}

func (e *Executor) data(stream *pipeline.TransferStream, client pipeline.Client) *model.PipelineError {
	stream.Transfer.Status = model.StatusTransfer
	if err := stream.Transfer.Update(e.Db); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewPipelineError(model.TeInternal, err.Error())
	}

	err := client.Data(stream)
	if err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
	}
	return err
}

func (e *Executor) runTransfer(stream *pipeline.TransferStream) {
	tErr := func() *model.PipelineError {
		client, cErr := e.getClient(stream)
		if cErr != nil {
			return cErr
		}

		if pErr := e.prologue(client); pErr != nil {
			return pErr
		}

		if pErr := stream.PreTasks(); pErr != nil {
			return pErr
		}
		if dErr := e.data(stream, client); dErr != nil {
			return dErr
		}
		if e := stream.Close(); e != nil {
			stream.Logger.Warningf("Failed to close local file: %s", e.Error())
		}
		if pErr := stream.PostTasks(); pErr != nil {
			return pErr
		}

		stream.Transfer.Status = model.StatusDone
		stream.Archive()
		stream.Exit()
		return nil
	}()

	if tErr != nil {
		pipeline.HandleError(stream, tErr)
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
				continue
			}
			e.runTransfer(stream)
		}
		wg.Done()
	}()
}
