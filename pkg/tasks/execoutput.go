package tasks

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// ExecOutputTask is a task which executes an external program which moves the
// transferred file.
type ExecOutputTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	RunnableTasks["EXECOUTPUT"] = &ExecOutputTask{}
	model.ValidTasks["EXECOUTPUT"] = &ExecOutputTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *ExecOutputTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
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
// FIXME: Should be refactored and factorized between all exec* tasks
//nolint:funlen // Should be refactored
func (e *ExecOutputTask) Run(params map[string]string, processor *Processor) (string, error) {
	path, args, delay, err := parseExecArgs(params)
	if err != nil {
		return err.Error(), fmt.Errorf("failed to parse task arguments: %w", err)
	}

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

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

	//nolint:errcheck // Those errors can be useless in a defered function
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
		return maxDelayExpired, errCommandTimeout
	default:
		var ex *exec.ExitError
		if ok := errors.As(cmdErr, &ex); ok && ex.ExitCode() == 1 {
			return cmdErr.Error(), errWarning
		}
	}

	//nolint:errcheck // This error can be useless at this stage
	_ = out.Close()

	msg, err := ioutil.ReadAll(in)
	if err != nil {
		return err.Error(), err
	}

	if newPath := getNewFileName(string(msg)); newPath != "" {
		if processor.Rule.IsSend {
			processor.Transfer.SourceFile = newPath
		} else {
			processor.Transfer.DestFile = newPath
		}
	}

	return string(msg), cmdErr
}
