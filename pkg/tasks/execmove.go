package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// execMoveTask is a task which executes an external program which moves the
// transferred file.
type execMoveTask struct{}

func init() {
	model.ValidTasks["EXECMOVE"] = &execMoveTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *execMoveTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

// Run executes the task by executing an external program with the given parameters.
func (e *execMoveTask) Run(params map[string]string, _ *database.DB,
	info *model.TransferContext, parent context.Context) (string, error) {
	script, args, delay, err := parseExecArgs(params)
	if err != nil {
		return "", fmt.Errorf("failed to parse task arguments: %s", err)
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if delay != 0 {
		ctx, cancel = context.WithTimeout(parent, delay)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}
	defer cancel()

	cmd := getCommand(ctx, script, args)
	in, out, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("max execution delay expired")
		default:
			if ex, ok := err.(*exec.ExitError); ok && ex.ExitCode() == 1 {
				return "", &errWarning{err.Error()}
			}
			return "", err
		}
	}
	_ = out.Close()

	scanner := bufio.NewScanner(in)
	var newPath string
	for scanner.Scan() {
		newPath = scanner.Text()
	}

	if _, err := os.Stat(newPath); err != nil {
		return "", fmt.Errorf("could not find moved file: %s", err)
	}

	info.Transfer.TrueFilepath = utils.NormalizePath(newPath)
	if info.Rule.IsSend {
		info.Transfer.SourceFile = path.Base(info.Transfer.TrueFilepath)
	} else {
		info.Transfer.DestFile = path.Base(info.Transfer.TrueFilepath)
	}
	return "", nil
}
