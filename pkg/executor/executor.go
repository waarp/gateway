// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"context"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
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

func (e *Executor) getClient(stream *pipeline.TransferStream) error {
	info, err := model.NewOutTransferInfo(e.DB, stream.Transfer)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer info: %s", err)
		return err
	}

	constr, ok := ClientsConstructors[info.Agent.Protocol]
	if !ok {
		msg := fmt.Sprintf("Unknown transfer protocol '%s'", info.Agent.Protocol)
		e.Logger.Critical(msg)

		return types.NewTransferError(types.TeUnimplemented, msg)
	}

	e.client, err = constr(*info, stream.Signals)
	if err != nil {
		msg := fmt.Sprintf("Failed to create transfer client: %s", err)
		e.Logger.Critical(msg)

		return types.NewTransferError(types.TeInternal, msg)
	}

	return nil
}

func (e *Executor) setup() error {
	e.Logger.Debug("Sending transfer request to remote server")

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

func (e *Executor) data() error {
	e.Logger.Debug("Starting data transfer")

	if e.TransferStream.Transfer.Step != types.StepPreTasks &&
		e.TransferStream.Transfer.Step != types.StepData {
		return nil
	}

	e.Transfer.Step = types.StepData
	e.Transfer.TaskNumber = 0
	if err := e.DB.Update(e.Transfer).Cols("step", "task_number").Run(); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return types.NewTransferError(types.TeInternal, err.Error())
	}

	if err := e.Start(); err != nil {
		return err
	}

	if err := e.client.Data(e.TransferStream); err != nil {
		_ = e.Close()
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return err
	}

	if err := e.Close(); err != nil {
		return err
	}
	if err := e.Move(); err != nil {
		return err
	}
	e.Logger.Debug("Data transfer done")

	return nil
}

func logTrans(logger *log.Logger, info *model.OutTransferInfo) {
	if info.Rule.IsSend {
		logger.Debugf("Starting %s upload of file '%s' to partner '%s' as '%s' using rule '%s'",
			info.Agent.Protocol, info.Transfer.SourceFile, info.Agent.Name,
			info.Account.Login, info.Rule.Name)
	} else {
		logger.Debugf("Starting %s download of file '%s' from partner '%s' as '%s' using rule '%s'",
			info.Agent.Protocol, info.Transfer.SourceFile, info.Agent.Name,
			info.Account.Login, info.Rule.Name)
	}
}

func (e *Executor) prologue() error {
	if e.Transfer.Step < types.StepSetup {
		e.Transfer.Step = types.StepSetup
	}

	if err := e.DB.Update(e.Transfer).Cols("step").Run(); err != nil {
		e.Logger.Criticalf("Failed to update transfer step to 'SETUP': %s", err)
		return err
	}

	if err := e.getClient(e.TransferStream); err != nil {
		e.Transfer.Step = types.StepSetup
		return err
	}

	if err := e.setup(); err != nil {
		e.Transfer.Step = types.StepSetup
		_ = e.client.Close(err)
		return err
	}

	return nil
}

func (e *Executor) run() error {
	info, err := model.NewOutTransferInfo(e.DB, e.Transfer)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer info: %s", err)
		return err
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

	e.Transfer.Step = types.StepFinalization
	e.Transfer.TaskNumber = 0
	if err := e.DB.Update(e.Transfer).Cols("step", "task_number").Run(); err != nil {
		e.Logger.Criticalf("Failed to update transfer step to '%s': %s",
			types.StepFinalization, err)
		return types.NewTransferError(types.TeInternal, "internal database error")
	}

	e.Logger.Debug("Sending transfer end message")
	if err := e.client.Close(nil); err != nil {
		e.Logger.Errorf("Remote post-task failed")
		return err
	}

	e.Transfer.Step = types.StepNone
	e.Transfer.Status = types.StatusDone

	return nil
}

// Run executes the transfer stream given in the executor.
func (e *Executor) Run() {
	e.Logger.Debugf("Processing transfer n°%d", e.Transfer.ID)

	if tErr := e.run(); tErr != nil {
		pipeline.HandleError(e.TransferStream, tErr)
		return
	}

	if e.Archive() == nil {
		e.Logger.Debug("Execution finished without errors")
	}
}
