// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"context"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

// ClientConstructor is the type representing the constructors used to make new
// instances of transfer clients. All transfer clients must have a ClientConstructor
// function in order to be called by the transfer executor.
type ClientConstructor func(model.OutTransferInfo, <-chan model.Signal) (pipeline.Client, error)

// ClientsConstructors is a map associating a protocol to its client constructor.
var ClientsConstructors = map[string]ClientConstructor{}

// Executor is the process responsible for executing outgoing transfers.
type Executor struct {
	*pipeline.TransferStream
	client  pipeline.Client
	Ctx     context.Context
	R66Home string
}

func (e *Executor) getClient(stream *pipeline.TransferStream) *model.PipelineError {
	info, err := model.NewOutTransferInfo(e.DB, stream.Transfer)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer info: %s", err)
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	constr, ok := ClientsConstructors[info.Agent.Protocol]
	if !ok {
		msg := fmt.Sprintf("Unknown transfer protocol '%s'", info.Agent.Protocol)
		e.Logger.Critical(msg)

		return model.NewPipelineError(model.TeUnimplemented, msg)
	}

	e.client, err = constr(*info, stream.Signals)
	if err != nil {
		msg := fmt.Sprintf("Failed to create transfer client: %s", err)
		e.Logger.Critical(msg)

		return model.NewPipelineError(model.TeInternal, msg)
	}

	return nil
}

func (e *Executor) setup() *model.PipelineError {
	e.Logger.Info("Sending transfer request to remote server '%s'")

	if err := e.client.Connect(); err != nil {
		e.Logger.Errorf("Failed to connect to remote server: %s", err)
		return err
	}

	if err := e.client.Authenticate(); err != nil {
		e.Logger.Errorf("Failed to authenticate on remote server: %s", err)
		return err
	}

	if err := e.client.Request(); err != nil {
		e.Logger.Errorf("Failed to make transfer request: %s", err)
		return err
	}

	return nil
}

func (e *Executor) data() *model.PipelineError {
	e.Logger.Info("Starting data transfer")

	if e.TransferStream.Transfer.Step != model.StepPreTasks &&
		e.TransferStream.Transfer.Step != model.StepData {
		return nil
	}

	e.Transfer.Step = model.StepData
	if err := e.DB.Update(e.Transfer); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewPipelineError(model.TeInternal, err.Error())
	}

	if err := e.Start(); err != nil {
		return err
	}

	if err := e.client.Data(e.TransferStream); err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return err
	}

	if err := e.Close(); err != nil {
		return err.(*model.PipelineError)
	}

	return nil
}

func logTrans(logger *log.Logger, info *model.OutTransferInfo) {
	if info.Rule.IsSend {
		logger.Infof("Starting %s upload of file '%s' to partner '%s' as '%s' using rule '%s'",
			info.Agent.Protocol, info.Transfer.SourceFile, info.Agent.Name,
			info.Account.Login, info.Rule.Name)
	} else {
		logger.Infof("Starting %s download of file '%s' from partner '%s' as '%s' using rule '%s'",
			info.Agent.Protocol, info.Transfer.SourceFile, info.Agent.Name,
			info.Account.Login, info.Rule.Name)
	}
}

func (e *Executor) prologue() *model.PipelineError {
	oldStep := model.StepSetup
	if e.Transfer.Step > model.StepSetup {
		oldStep = e.Transfer.Step
	}

	e.Transfer.Step = model.StepSetup

	defer func() { e.Transfer.Step = oldStep }()

	e.Transfer.Status = model.StatusRunning

	if err := e.DB.Update(e.Transfer); err != nil {
		e.Logger.Criticalf("Failed to update transfer step to 'SETUP': %s", err)
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	if err := e.getClient(e.TransferStream); err != nil {
		return err
	}

	if err := e.setup(); err != nil {
		_ = e.client.Close(err)

		return err
	}

	return nil
}

func (e *Executor) run() *model.PipelineError {
	info, err := model.NewOutTransferInfo(e.DB, e.Transfer)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer info: %s", err)
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	logTrans(e.Logger, info)

	if err := e.prologue(); err != nil {
		return err
	}

	if err := e.TransferStream.PreTasks(); err != nil {
		_ = e.client.Close(err)
		return err
	}

	if err := e.data(); err != nil {
		_ = e.client.Close(err)
		return err
	}

	if err := e.TransferStream.PostTasks(); err != nil {
		_ = e.client.Close(err)
		return err
	}

	if err := e.client.Close(nil); err != nil {
		e.Logger.Errorf("Remote post-task failed")
		return err
	}

	e.TransferStream.Transfer.Step = model.StepNone
	e.TransferStream.Transfer.Status = model.StatusDone

	return nil
}

// Run executes the transfer stream given in the executor.
func (e *Executor) Run() {
	e.Logger.Infof("Processing transfer n°%d", e.Transfer.ID)

	if tErr := e.run(); tErr != nil {
		pipeline.HandleError(e.TransferStream, tErr)
		return
	}

	if e.Archive() == nil {
		e.Logger.Info("Execution finished without errors")
	}
}
