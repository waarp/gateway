package tasks

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// ExecMoveTask is a task which executes an external program which moves the
// transferred file.
type ExecMoveTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	RunnableTasks["EXECMOVE"] = &ExecMoveTask{}
	model.ValidTasks["EXECMOVE"] = &ExecMoveTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *ExecMoveTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

// Run executes the task by executing an external program with the given parameters.
// FIXME: Should be refactored and factorized between all exec* tasks
//nolint:funlen // Should be refactored
func (e *ExecMoveTask) Run(params map[string]string, processor *Processor) (string, error) {
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

	if err := cmd.Run(); err != nil {
		select {
		case <-ctx.Done():
			return maxDelayExpired, errCommandTimeout
		default:
			var ex *exec.ExitError
			if ok := errors.As(err, &ex); ok && ex.ExitCode() == 1 {
				return err.Error(), errWarning
			}

			return err.Error(), err
		}
	}

	//nolint:errcheck // Those errors can be useless in this case
	_ = out.Close()

	var newPath string

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		newPath = scanner.Text()
	}

	if _, err := os.Stat(newPath); err != nil {
		return "could not find moved file", fmt.Errorf("could not find moved file: %w", err)
	}

	if processor.Rule.IsSend {
		processor.Transfer.SourceFile = newPath
	} else {
		processor.Transfer.DestFile = newPath
	}

	return "", nil
}
