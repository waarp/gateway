package tasks

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// execTask is a task which executes an external program.
type execTask struct{}

func init() {
	model.ValidTasks["EXEC"] = &execTask{}
}

func parseExecArgs(params map[string]string) (path, args string,
	delay time.Duration, err error) {

	var ok bool
	if path, ok = params["path"]; !ok || path == "" {
		err = fmt.Errorf("missing program path")
		return
	}

	if args, ok = params["args"]; !ok {
		err = fmt.Errorf("missing program arguments")
		return
	}

	d, ok := params["delay"]
	if !ok {
		err = fmt.Errorf("missing program delay")
		return
	}
	d2, err := strconv.ParseFloat(d, 64)
	if err != nil {
		err = fmt.Errorf("invalid program delay")
		return
	}
	if delay < 0 {
		err = fmt.Errorf("invalid program delay value (must be positive or 0)")
		return
	}
	delay = time.Duration(d2) * time.Millisecond
	return
}

// Validate checks if the EXEC task has all the required arguments.
func (e *execTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

// Run executes the task by executing the external program with the given parameters.
func (e *execTask) Run(params map[string]string, _ *database.DB,
	_ *model.TransferContext, parent context.Context) (string, error) {

	path, args, delay, err := parseExecArgs(params)
	if err != nil {
		return "", fmt.Errorf("failed to parse task arguments: %s", err)
	}

	ctx := parent
	var cancel context.CancelFunc
	if delay != 0 {
		ctx, cancel = context.WithTimeout(parent, delay)
		defer cancel()
	}

	cmd := getCommand(ctx, path, args)

	execErr := cmd.Run()
	if execErr == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("max execution delay expired")
	default:
		if ex, ok := execErr.(*exec.ExitError); ok && ex.ExitCode() == 1 {
			return "", &errWarning{execErr.Error()}
		}
		return "", execErr
	}
}
