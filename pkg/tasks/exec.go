package tasks

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// execTask is a task which executes an external program.
type execTask struct{}

// Validate checks if the EXEC task has all the required arguments.
func (e *execTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

// Run executes the task by executing the external program with the given parameters.
func (e *execTask) Run(parent context.Context, params map[string]string,
	_ *database.DB, logger *log.Logger, _ *model.TransferContext,
) error {
	output, cmdErr := runExec(parent, params)
	if cmdErr != nil {
		return cmdErr
	}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		logger.Debug(scanner.Text())
	}

	logger.Debugf("Done executing command %s %s", params["path"], params["args"])

	return nil
}

//nolint:revive //need multiple return results here
func parseExecArgs(params map[string]string) (path, args string,
	delay time.Duration, err error,
) {
	var hasPath bool
	if path, hasPath = params["path"]; !hasPath || path == "" {
		err = fmt.Errorf("missing program path: %w", ErrBadTaskArguments)

		return "", "", 0, err
	}

	args = params["args"]

	if delayStr, hasDelay := params["delay"]; hasDelay {
		//nolint:mnd // useless to define a constant
		delayMs, parsErr := strconv.ParseInt(delayStr, 10, 64)
		if parsErr != nil {
			parsErr = fmt.Errorf("invalid program delay: %w", ErrBadTaskArguments)

			return "", "", 0, parsErr
		}

		if delayMs < 0 {
			err = fmt.Errorf("invalid program delay value (must be positive or 0): %w",
				ErrBadTaskArguments)

			return "", "", 0, err
		}

		delay = time.Duration(delayMs) * time.Millisecond
	}

	return path, args, delay, nil
}

func runExec(parent context.Context, params map[string]string) (*bytes.Buffer, error) {
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
	cmd.Stdout = &output

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

		return nil, ErrCommandTimeout

	case cmdErr := <-waitDone:
		var ex *exec.ExitError
		if ok := errors.As(cmdErr, &ex); ok && ex.ExitCode() == 1 {
			return &output, &WarningError{cmdErr.Error()}
		}

		return &output, cmdErr
	}
}
