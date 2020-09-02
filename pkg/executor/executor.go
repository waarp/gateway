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

	e.TransferStream.Transfer.Step = model.StepData
	if err := e.TransferStream.Transfer.Update(e.DB); err != nil {
		e.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewPipelineError(model.TeInternal, err.Error())
	}

	if err := e.TransferStream.Start(); err != nil {
		return err
	}

	if err := e.client.Data(e.TransferStream); err != nil {
		e.Logger.Errorf("Error while transmitting data: %s", err)
		return err
	}

	if err := e.TransferStream.Close(); err != nil {
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

	if err := e.Transfer.Update(e.DB); err != nil {
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

	if info.Agent.Protocol == "r66" {
		e.runR66(info)
		return nil
	}

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

func (e *Executor) runR66(info *model.OutTransferInfo) {
	if err := e.r66Transfer(info); err != nil {
		msg := fmt.Sprintf("Transfer failed: %s", err)
		e.Logger.Error(msg)
		e.Transfer.Status = model.StatusError
	} else {
		e.Transfer.Status = model.StatusDone
	}
}

//nolint:funlen,nestif,gomnd,goerr113 // temporary function that will eventually be removed
func (e *Executor) r66Transfer(info *model.OutTransferInfo) error {
	e.Logger.Infof("Delegating R66 transfer n°%d to external server", e.Transfer.ID)
	script := e.R66Home
	args := buildR66CommandArgs(info)

	e.Logger.Debugf("%s %#v", script, args)
	cmd := exec.Command(script, args...) //nolint:gosec //args has already been sanitized

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
		if err2 := json.Unmarshal(arrays[1], result); err2 != nil {
			return err2
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

		buf, err3 := json.Marshal(result)
		if err3 != nil {
			return err3
		}

		info.Transfer.ExtInfo = buf
	}

	return err
}

func buildR66CommandArgs(info *model.OutTransferInfo) []string {
	return []string{
		info.Account.Login,
		"send",
		"-to", info.Agent.Name,
		"-file", info.Transfer.SourceFile,
		"-rule", info.Rule.Name,
	}
}

type r66Result struct {
	SpecialID       int
	StatusCode      string
	StatusTxt       string
	FinalPath       string
	FileInformation string
	OriginalSize    uint
}
