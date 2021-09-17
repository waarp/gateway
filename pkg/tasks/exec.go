package tasks

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const maxDelayExpired = "max exec delay expired"

// ExecTask is a task which executes an external program.
type ExecTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	RunnableTasks["EXEC"] = &ExecTask{}
	model.ValidTasks["EXEC"] = &ExecTask{}
}

func parseExecArgs(params map[string]string) (path, args string,
	delay float64, err error) {
	var ok bool
	if path, ok = params["path"]; !ok || path == "" {
		err = fmt.Errorf("missing program path: %w", errBadTaskArguments)

		return
	}

	if args, ok = params["args"]; !ok {
		err = fmt.Errorf("missing program arguments: %w", errBadTaskArguments)

		return
	}

	d, ok := params["delay"]
	if !ok {
		err = fmt.Errorf("missing program delay: %w", errBadTaskArguments)

		return
	}

	delay, err = strconv.ParseFloat(d, 64) //nolint:gomnd // useless to define a constant
	if err != nil {
		err = fmt.Errorf("invalid program delay: %w", errBadTaskArguments)

		return
	}

	if delay < 0 {
		err = fmt.Errorf("invalid program delay value (must be positive or 0): %w",
			errBadTaskArguments)

		return
	}

	return path, args, delay, nil
}

// Validate checks if the EXEC task has all the required arguments.
func (e *ExecTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

// Run executes the task by executing the external program with the given parameters.
func (e *ExecTask) Run(params map[string]string, _ *Processor) (string, error) {
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

	execErr := cmd.Run() //nolint:ifshort // false positive
	if execErr == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return maxDelayExpired, errCommandTimeout

	default:
		var ex *exec.ExitError
		if ok := errors.As(execErr, &ex); ok && ex.ExitCode() == 1 {
			return execErr.Error(), errWarning
		}

		return execErr.Error(), execErr
	}
}
