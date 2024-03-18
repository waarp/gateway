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
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
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
func (e *execMoveTask) Run(parent context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
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

	newURL, err := types.ParseURL(newPath)
	if err != nil {
		return fmt.Errorf("failed to parse the new file path %q: %w", newPath, err)
	}

	newFS, fsErr := fs.GetFileSystem(db, newURL)
	if fsErr != nil {
		return fmt.Errorf("failed to instantiate filesystem for new file %q: %w", newPath, fsErr)
	}

	if _, err := fs.Stat(newFS, newURL); err != nil {
		return fmt.Errorf("could not find moved file %q: %w", newPath, err)
	}

	transCtx.FS = newFS
	transCtx.Transfer.LocalPath = *newURL
	transCtx.Transfer.RemotePath = path.Join(
		path.Dir(transCtx.Transfer.RemotePath), path.Base(newURL.Path))

	logger.Debug("Done executing command %s %s", params["path"], params["args"])

	return nil
}
