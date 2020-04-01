package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// ExecTask is a task which executes an external program.
type ExecTask struct{}

func init() {
	RunnableTasks["EXEC"] = &ExecTask{}
}

func parseExecArgs(params map[string]interface{}) (path, args string,
	delay float64, err error) {

	var ok bool
	if path, ok = params["path"].(string); !ok || path == "" {
		err = fmt.Errorf("missing program path")
		return
	}
	if args, ok = params["args"].(string); !ok {
		err = fmt.Errorf("missing program arguments")
		return
	}
	if delay, ok = params["delay"].(float64); !ok || delay < 0 {
		err = fmt.Errorf("missing program delay")
		return
	}
	return
}

// Validate checks if the EXEC task has all the required arguments.
func (e *ExecTask) Validate(task *model.Task) error {
	var params map[string]interface{}
	if err := json.Unmarshal(task.Args, &params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

// Run executes the task by executing the external program with the given parameters.
func (e *ExecTask) Run(params map[string]interface{}, processor *Processor) (string, error) {

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

	execErr := cmd.Run()
	if execErr == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "max exec delay expired", fmt.Errorf("max exec delay expired")
	default:
		if ex, ok := execErr.(*exec.ExitError); ok && ex.ExitCode() == 1 {
			return execErr.Error(), errWarning
		}
		return execErr.Error(), execErr
	}
}
