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

// Validate checks if the EXEC task has all the required arguments.
func (e *ExecTask) Validate(task *model.Task) error {
	var args map[string]interface{}
	if err := json.Unmarshal(task.Args, &args); err != nil {
		return err
	}

	if path, ok := args["path"].(string); !ok || path == "" {
		return fmt.Errorf("missing program path")
	}
	if _, ok := args["args"].(string); !ok {
		return fmt.Errorf("missing program arguments")
	}
	if delay, ok := args["delay"].(float64); !ok || delay == 0 {
		return fmt.Errorf("missing program delay")
	}

	return nil
}

// Run executes the task by executing the external program with the given parameters.
func (e *ExecTask) Run(params map[string]interface{}, processor *Processor) (string, error) {
	var path, args string
	var delay float64
	var ok bool
	if path, ok = params["path"].(string); !ok || path == "" {
		return "missing program path", fmt.Errorf("missing program path")
	}
	if args, ok = params["args"].(string); !ok {
		return "missing program arguments", fmt.Errorf("missing program arguments")
	}
	if delay, ok = params["delay"].(float64); !ok || delay == 0 {
		return "missing program delay", fmt.Errorf("missing program delay")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(delay)*
		time.Millisecond)
	defer cancel()

	cmd := getCommand(ctx, path, args)

	err := cmd.Run()
	if err == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "max exec delay expired", fmt.Errorf("max exec delay expired")
	default:
		if ex, ok := err.(*exec.ExitError); ok && ex.ExitCode() == 1 {
			return err.Error(), errWarning
		}
		return err.Error(), err
	}
}
