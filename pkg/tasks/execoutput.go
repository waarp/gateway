package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// execOutputTask is a task which executes an external program which moves the
// transferred file.
type execOutputTask struct{}

func init() {
	model.ValidTasks["EXECOUTPUT"] = &execOutputTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *execOutputTask) Validate(params map[string]string) error {

	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

func getNewFileName(output string) string {
	lines := strings.Split(strings.TrimSpace(output), lineSeparator)
	lastLine := lines[len(lines)-1]

	if strings.HasPrefix(lastLine, "NEWFILENAME:") {
		return strings.TrimPrefix(lastLine, "NEWFILENAME:")
	}
	return ""
}

// Run executes the task by executing an external program with the given parameters.
func (e *execOutputTask) Run(params map[string]string, _ *database.DB,
	transCtx *model.TransferContext, parent context.Context) (string, error) {

	script, args, delay, err := parseExecArgs(params)
	if err != nil {
		return "", fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	ctx := parent
	var cancel context.CancelFunc
	if delay != 0 {
		ctx, cancel = context.WithTimeout(parent, delay)
		defer cancel()
	}

	cmd := getCommand(ctx, script, args)
	in, out, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("failed to pipe script output: %s", err)
	}
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	cmd.Stdout = out

	cmdErr := cmd.Run()
	_ = out.Close()
	msg, err := ioutil.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("failed to read script output: %s", err)
	}

	if cmdErr != nil {
		if newPath := getNewFileName(string(msg)); newPath != "" {
			transCtx.Transfer.LocalPath = utils.ToOSPath(newPath)
			transCtx.Transfer.RemotePath = utils.ToStandardPath(path.Dir(
				transCtx.Transfer.RemotePath), path.Base(transCtx.Transfer.LocalPath))
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("max execution delay expired")
		default:
			if ex, ok := cmdErr.(*exec.ExitError); ok && ex.ExitCode() == 1 {
				return "", &errWarning{cmdErr.Error()}
			}
			return "", cmdErr
		}
	}

	return string(msg), nil
}
