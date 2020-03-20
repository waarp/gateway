package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// ExecMoveTask is a task which executes an external program which moves the
// transferred file.
type ExecMoveTask struct{}

func init() {
	RunnableTasks["EXECMOVE"] = &ExecMoveTask{}
	model.ValidTasks["EXECMOVE"] = &ExecMoveTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *ExecMoveTask) Validate(args map[string]string) error {
	if path, ok := args["path"]; !ok || path == "" {
		return fmt.Errorf("missing program path")
	}
	if _, ok := args["args"]; !ok {
		return fmt.Errorf("missing program arguments")
	}
	delay, ok := args["delay"]
	if !ok {
		return fmt.Errorf("missing program delay")
	}
	if _, err := strconv.ParseInt(delay, 10, 32); err != nil {
		return fmt.Errorf("delay must be an integer")
	}

	return nil
}

// Run executes the task by executing an external program with the given parameters.
func (e *ExecMoveTask) Run(params map[string]interface{}, processor *Processor) (string, error) {
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
	in, out, err := os.Pipe()
	if err != nil {
		return err.Error(), err
	}
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
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
	_ = out.Close()

	scanner := bufio.NewScanner(in)
	var newPath string
	for scanner.Scan() {
		newPath = scanner.Text()
	}

	if _, err := os.Stat(newPath); err != nil {
		return "could not find moved file", err
	}

	if processor.Rule.IsSend {
		processor.Transfer.SourcePath = newPath
	} else {
		processor.Transfer.DestPath = newPath
	}
	return "", nil
}
