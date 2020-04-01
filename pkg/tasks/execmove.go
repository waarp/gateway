package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
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
func (e *ExecMoveTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

// Run executes the task by executing an external program with the given parameters.
func (e *ExecMoveTask) Run(params map[string]string, processor *Processor) (string, error) {
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
