package tasks

import (
	"bufio"
	"context"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// execMoveTask is a task which executes an external program which moves the
// transferred file.
type execMoveTask struct{}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *execMoveTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

// Run executes the task by executing an external program with the given parameters.
func (e *execMoveTask) Run(ctx context.Context, params map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	output, cmdErr := runExec(ctx, transCtx, params)
	if cmdErr != nil {
		return cmdErr
	}

	var newPath string

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		newPath = scanner.Text()
		logger.Debug(newPath)
	}

	if _, err := fs.Stat(newPath); err != nil {
		return fmt.Errorf("could not find moved file %q: %w", newPath, err)
	}

	transCtx.Transfer.LocalPath = newPath
	transCtx.Transfer.RemotePath = path.Join(
		path.Dir(transCtx.Transfer.RemotePath), path.Base(newPath))

	logger.Debugf("Done executing command %s %s", params["path"], params["args"])

	return nil
}
