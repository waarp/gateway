package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// execMoveTask is a task which executes an external program which moves the
// transferred file.
type execMoveTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["EXECMOVE"] = &execMoveTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *execMoveTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

// Run executes the task by executing an external program with the given parameters.
func (e *execMoveTask) Run(parent context.Context, params map[string]string, _ *database.DB, logger *log.Logger, transCtx *model.TransferContext) error {
	output, cmdErr := runExec(parent, params)
	if cmdErr != nil {
		return cmdErr
	}

	var newPath string

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		newPath = scanner.Text()
		logger.Debug(newPath)
	}

	if _, err := os.Stat(newPath); err != nil {
		return fmt.Errorf("could not find moved file: %w", err)
	}

	transCtx.Transfer.LocalPath = utils.ToOSPath(newPath)
	transCtx.Transfer.RemotePath = path.Join(path.Dir(transCtx.Transfer.RemotePath),
		path.Base(transCtx.Transfer.LocalPath))

	logger.Debug("Done executing command %s %s", params["path"], params["args"])

	return nil
}
