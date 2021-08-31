package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
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
func (e *execMoveTask) Run(parent context.Context, params map[string]string,
	_ *database.DB, transCtx *model.TransferContext) (string, error) {

	output, cmdErr := runExec(parent, params, true)
	if cmdErr != nil {
		return "", cmdErr
	}

	scanner := bufio.NewScanner(output)
	var newPath string
	for scanner.Scan() {
		newPath = scanner.Text()
	}

	if _, err := os.Stat(newPath); err != nil {
		return "", fmt.Errorf("could not find moved file: %s", err)
	}

	transCtx.Transfer.LocalPath = utils.ToStandardPath(newPath)
	transCtx.Transfer.RemotePath = path.Join(path.Dir(transCtx.Transfer.RemotePath),
		path.Base(transCtx.Transfer.LocalPath))

	return "", nil
}
