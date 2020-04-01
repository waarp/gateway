package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// ExecOutputTask is a task which executes an external program which moves the
// transferred file.
type ExecOutputTask struct{}

func init() {
	RunnableTasks["EXECOUTPUT"] = &ExecOutputTask{}
	model.ValidTasks["EXECOUTPUT"] = &ExecOutputTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *ExecOutputTask) Validate(params map[string]string) error {

	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

func getNewFileName(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := lines[len(lines)-1]

	if strings.HasPrefix(lastLine, "NEWFILENAME:") {
		return strings.TrimPrefix(lastLine, "NEWFILENAME:")
	}
	return ""
}

// Run executes the task by executing an external program with the given parameters.
func (e *ExecOutputTask) Run(params map[string]string, processor *Processor) (string, error) {

	path, args, delay, err := parseExecArgs(params)
	if err != nil {
		return err.Error(), fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if delay != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(delay)*
			time.Millisecond)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	cmd := getCommand(ctx, path, args)
	in, out, err := os.Pipe()
	if err != nil {
		return err.Error(), err
	}
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	cmd.Stdout = out

	cmdErr := cmd.Run()
	if cmdErr == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "max exec delay expired", fmt.Errorf("max exec delay expired")
	default:
		if ex, ok := cmdErr.(*exec.ExitError); ok && ex.ExitCode() == 1 {
			return cmdErr.Error(), errWarning
		}
	}

	_ = out.Close()
	msg, err := ioutil.ReadAll(in)
	if err != nil {
		return err.Error(), err
	}

	if newPath := getNewFileName(string(msg)); newPath != "" {
		if processor.Rule.IsSend {
			processor.Transfer.SourcePath = newPath
		} else {
			processor.Transfer.DestPath = newPath
		}
	}
	return string(msg), cmdErr
}
