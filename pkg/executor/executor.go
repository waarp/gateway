// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

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

func (e *Executor) getClient(stream *pipeline.TransferStream) (te *model.PipelineError) {
	info, err := model.NewOutTransferInfo(e.DB, stream.Transfer)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer info: %s", err)
		te = &model.PipelineError{Kind: model.KindDatabase}
		return
	}

	constr, ok := ClientsConstructors[info.Agent.Protocol]
	if !ok {
		msg := fmt.Sprintf("Unknown transfer protocol '%s'", info.Agent.Protocol)
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
		if err.Cause.Code == model.TeExternalOperation {
			e.TransferStream.Transfer.Step = model.StepPreTasks
		}
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

	e.TransferStream.Transfer.Step = model.StepData
	if err := e.TransferStream.Transfer.Update(e.DB); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewPipelineError(model.TeInternal, err.Error())
	}

	if err := e.client.Data(e.TransferStream); err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return err
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

// Run executes the transfer stream given in the executor.
func (e *Executor) Run() {
	e.Logger.Infof("Processing transfer n°%d", e.Transfer.ID)

	var tErr *model.PipelineError
	info, err := model.NewOutTransferInfo(e.DB, e.Transfer)
	if err != nil {
		e.Logger.Criticalf("Failed to retrieve transfer info: %s", err)
		tErr = &model.PipelineError{Kind: model.KindDatabase}
		pipeline.HandleError(e.TransferStream, tErr)
		return
	}
	logTrans(e.Logger, info)
	if info.Agent.Protocol == "r66" {
		e.runR66(info)
		return
	}

	tErr = func() *model.PipelineError {
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

		e.TransferStream.Transfer.Status = model.StatusDone
		e.TransferStream.Archive()
		e.TransferStream.Exit()
		return nil
	}()

	if tErr != nil {
		pipeline.HandleError(e.TransferStream, tErr)
	}
	e.Logger.Infof("Transfer n°%d finished without errors", e.Transfer.ID)
}

func (e *Executor) runR66(info *model.OutTransferInfo) {
	if err := e.r66Transfer(info); err != nil {
		msg := fmt.Sprintf("Transfer failed: %s", err)
		e.Logger.Error(msg)
		e.Transfer.Status = model.StatusError
	} else {
		e.Transfer.Status = model.StatusDone
	}
	e.Pipeline.Archive()
	e.Pipeline.Exit()
}

func (e *Executor) r66Transfer(info *model.OutTransferInfo) error {
	e.Logger.Infof("Delegating R66 transfer n°%d to external server", e.Transfer.ID)
	script := e.R66Home
	args := []string{
		"send",
		info.Account.Login,
		"-to", info.Agent.Name,
		"-file", info.Transfer.SourceFile,
		"-rule", info.Rule.Name,
	}
	e.Logger.Debugf("%s %#v", script, args)
	cmd := exec.Command(script, args...) //nolint:gosec
	out, err := cmd.Output()
	defer func() {
		e.Logger.Debug("R66 server output:")
		for _, l := range bytes.Split(out, []byte{'\n'}) {
			e.Logger.Debugf("    %s", string(l))
		}
	}()
	if err != nil {
		info.Transfer.Error = model.TransferError{
			Code:    model.TeExternalOperation,
			Details: err.Error(),
		}
		return err
	}
	if len(out) > 0 {
		// Get the second line of the output
		arrays := bytes.Split(out, []byte("\n"))
		if len(arrays) < 2 {
			return fmt.Errorf("bad output")
		}
		// Parse into a r66Result
		result := &r66Result{}
		if err := json.Unmarshal(arrays[1], result); err != nil {
			return err
		}
		if len(result.StatusCode) == 0 {
			return fmt.Errorf("bad output")
		}

		e.Logger.Infof("R66 transfer finished with status code %s",
			result.StatusCode)
		// Add R66 result info to the transfer
		info.Transfer.Error.Code = model.FromR66Code(result.StatusCode[0])
		if info.Transfer.Error.Code != model.TeOk {
			info.Transfer.Error.Details = result.StatusTxt
		}
		info.Transfer.DestFile = result.FinalPath
		buf, err := json.Marshal(result)
		if err != nil {
			return err
		}
		info.Transfer.ExtInfo = buf
	}
	return err
}

type r66Result struct {
	SpecialID       int
	StatusCode      string
	StatusTxt       string
	FinalPath       string
	FileInformation string
	OriginalSize    uint
}
