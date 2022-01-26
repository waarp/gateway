package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// execTask is a task which executes an external program.
type execTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["EXEC"] = &execTask{}
}

// Validate checks if the EXEC task has all the required arguments.
func (e *execTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

// Run executes the task by executing the external program with the given parameters.
func (e *execTask) Run(parent context.Context, params map[string]string,
	_ *database.DB, _ *model.TransferContext) (string, error) {
	_, cmdErr := runExec(parent, params, false)

	return "", cmdErr
}

func parseExecArgs(params map[string]string) (path, args string,
	delay time.Duration, err error) {
	var ok bool
	if path, ok = params["path"]; !ok || path == "" {
		err = fmt.Errorf("missing program path: %w", errBadTaskArguments)

		return "", "", 0, err
	}

	if args, ok = params["args"]; !ok {
		err = fmt.Errorf("missing program arguments: %w", errBadTaskArguments)

		return "", "", 0, err
	}

	d, ok := params["delay"]
	if !ok {
		err = fmt.Errorf("missing program delay: %w", errBadTaskArguments)

		return "", "", 0, err
	}

	d2, err := strconv.ParseFloat(d, 64) //nolint:gomnd // useless to define a constant
	if err != nil {
		err = fmt.Errorf("invalid program delay: %w", errBadTaskArguments)

		return "", "", 0, err
	}

	if delay < 0 {
		err = fmt.Errorf("invalid program delay value (must be positive or 0): %w",
			errBadTaskArguments)

		return "", "", 0, err
	}

	delay = time.Duration(d2) * time.Millisecond

	return path, args, delay, nil
}

func runExec(parent context.Context, params map[string]string, withOutput bool) (*bytes.Buffer, error) {
	var (
		cancel context.CancelFunc
		output bytes.Buffer
	)

	script, args, delay, err := parseExecArgs(params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task arguments: %w", err)
	}

	ctx := parent

	if delay != 0 {
		ctx, cancel = context.WithTimeout(parent, delay)
		defer cancel()
	}

	cmd := getCommand(script, args)

	if withOutput {
		cmd.Stdout = &output
	}

	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("failed to start external program: %w", startErr)
	}

	waitDone := make(chan error)

	go func() {
		defer close(waitDone)

		waitDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if haltErr := haltExec(cmd, ctx); haltErr != nil {
			return nil, haltErr
		}

		<-waitDone

		return nil, errCommandTimeout

	case cmdErr := <-waitDone:
		var ex *exec.ExitError
		if ok := errors.As(cmdErr, &ex); ok && ex.ExitCode() == 1 {
			return &output, &warningError{cmdErr.Error()}
		}

		return &output, cmdErr
	}
}
