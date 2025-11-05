package tasks

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/google/shlex"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var (
	ErrExecTimeout   = errors.New("external program timed out")
	ErrExecCancelled = errors.New("external program was cancelled")
)

const execWaitDelay = 10 * time.Second

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
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	output, runErr := runExec(parent, logger, transCtx, params)
	if runErr != nil {
		return runErr
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

func makeCmd(transCtx *model.TransferContext, script, args string) (*exec.Cmd, error) {
	cmd, cmdErr := getCommand(script, args)
	if cmdErr != nil {
		return nil, cmdErr
	}

	cmd.Env = os.Environ()
	cmd.WaitDelay = execWaitDelay

	for key, replaceFunc := range getReplacers() {
		value, err := replaceFunc(transCtx, key)
		if errors.Is(err, errNotImplemented) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("failed to get replacement value %s: %w", key, err)
		}

		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	return cmd, nil
}

func runCmd(ctx context.Context, cmd *exec.Cmd, timeout time.Duration, logger *log.Logger,
) (stdout, stderr *bytes.Buffer, err error) {
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if startErr := cmd.Start(); startErr != nil {
		return stdout, stderr, fmt.Errorf("failed to start external program: %w", startErr)
	}

	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	result := make(chan error)
	go func() {
		defer close(result)
		result <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if hErr := haltExec(cmd); hErr != nil {
			logger.Warningf("failed to halt external program: %v", hErr)
		}
		<-result

		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return stdout, stderr, ErrExecTimeout
		case errors.Is(ctx.Err(), context.Canceled):
			return stdout, stderr, ErrExecCancelled
		default:
			return stdout, stderr, fmt.Errorf("program context error: %w", ctx.Err())
		}
	case runErr := <-result:
		var ex *exec.ExitError
		if ok := errors.As(runErr, &ex); ok && ex.ExitCode() == 1 {
			return stdout, stderr, &WarningError{runErr.Error()}
		}

		return stdout, stderr, runErr
	}
}

func getCommand(path, argsStr string) (*exec.Cmd, error) {
	args, err := shlex.Split(argsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid program arguments: %w", err)
	}

	return exec.Command(path, args...), nil
}

func runExec(ctx context.Context, logger *log.Logger, transCtx *model.TransferContext,
	params map[string]string,
) (*bytes.Buffer, error) {
	script, args, delay, parsErr := parseExecArgs(params)
	if parsErr != nil {
		return nil, fmt.Errorf("failed to parse task arguments: %w", parsErr)
	}

	cmd, cmdErr := makeCmd(transCtx, script, args)
	if cmdErr != nil {
		return nil, cmdErr
	}

	stdout, stderr, runErr := runCmd(ctx, cmd, delay, logger)
	switch {
	case errors.Is(runErr, context.DeadlineExceeded):
		return stdout, ErrExecTimeout
	case errors.Is(runErr, context.Canceled):
		return stdout, ErrExecCancelled
	case runErr != nil:
		if msg := stderr.String(); msg != "" {
			logger.Errorf("Program returned error: %s", msg)
		}

		return stdout, runErr
	default:
		return stdout, nil
	}
}
