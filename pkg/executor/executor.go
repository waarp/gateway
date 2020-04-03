// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"context"
	"fmt"

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
	*pipeline.TransferStream
	client  pipeline.Client
	Ctx     context.Context
	R66Home string
}

func (e *Executor) getClient(stream *pipeline.TransferStream) (te *model.PipelineError) {
	if oldStep := stream.Transfer.Step; oldStep != "" {
		defer func() {
			if te == nil {
				stream.Transfer.Step = oldStep
			}
		}()
	}
	stream.Transfer.Step = model.StepSetup
	if err := stream.Transfer.Update(stream.Db); err != nil {
		e.Logger.Criticalf("Failed to update transfer step: %s", err)
		te = &model.PipelineError{Kind: model.KindDatabase}
		return
	}

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
	e.client, err = constr(*info, stream.Signals)
	if err != nil {
		msg := fmt.Sprintf("Failed to create transfer client: %s", err)
		e.Logger.Critical(msg)
		te = model.NewPipelineError(model.TeConnection, msg)
		return
	}

	return te
}

func (e *Executor) prologue() *model.PipelineError {
	if err := e.client.Connect(); err != nil {
		msg := fmt.Sprintf("Failed to connect to remote agent: %s", err)
		e.Logger.Error(msg)
		return err
	}

	if err := e.client.Authenticate(); err != nil {
		msg := fmt.Sprintf("Failed to authenticate on remote agent: %s", err)
		e.Logger.Error(msg)
		return err
	}

	if err := e.client.Request(); err != nil {
		msg := fmt.Sprintf("Failed to make transfer request: %s", err)
		e.Logger.Error(msg)
		if err.Cause.Code == model.TeExternalOperation {
			e.TransferStream.Transfer.Step = model.StepPreTasks
		}
		return err
	}

	return nil
}

func (e *Executor) data() *model.PipelineError {
	if e.TransferStream.Transfer.Step != model.StepPreTasks && e.TransferStream.Transfer.Step != model.StepData {
		return nil
	}

	e.TransferStream.Transfer.Step = model.StepData
	if err := e.TransferStream.Transfer.Update(e.Db); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewPipelineError(model.TeInternal, err.Error())
	}

	if err := e.client.Data(e.TransferStream); err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return err
	}
	return nil
}

// Run executes the transfer stream given in the executor.
func (e *Executor) Run() {
	tErr := func() *model.PipelineError {
		if sErr := e.TransferStream.Start(); sErr != nil {
			return sErr
		}

		gErr := e.getClient(e.TransferStream)
		if gErr != nil {
			return gErr
		}

		if pErr := e.prologue(); pErr != nil {
			_ = e.client.Close(pErr)
			return pErr
		}

		if pErr := e.TransferStream.PreTasks(); pErr != nil {
			_ = e.client.Close(pErr)
			return pErr
		}
		if dErr := e.data(); dErr != nil {
			_ = e.client.Close(dErr)
			return dErr
		}
		if pErr := e.TransferStream.PostTasks(); pErr != nil {
			_ = e.client.Close(pErr)
			return pErr
		}
		if cErr := e.client.Close(nil); cErr != nil {
			e.Logger.Errorf("Remote post-task failed")
			return cErr
		}
		if fErr := e.TransferStream.Finalize(); fErr != nil {
			return fErr
		}

		e.TransferStream.Transfer.Status = model.StatusDone
		e.TransferStream.Archive()
		e.TransferStream.Exit()
		return nil
	}()

	if tErr != nil {
		pipeline.HandleError(e.TransferStream, tErr)
	}
}
