package tasks

import (
	"context"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// renameTask is a task which rename the target of the transfer
// without impacting the filesystem.
type renameTask struct{}

var ErrRenameMissingPath = fmt.Errorf(
	`cannot create a RENAME task without a "path" argument: %w`, ErrBadTaskArguments)

// Validate checks if the RENAME tasks has all the required arguments.
func (*renameTask) Validate(args map[string]string) error {
	if args["path"] == "" {
		return ErrRenameMissingPath
	}

	return nil
}

// Run executes the task by renaming the transfer file.
func (*renameTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newPath := args["path"]
	if newPath == "" {
		return ErrRenameMissingPath
	}

	if _, err := fs.Stat(newPath); err != nil {
		return fmt.Errorf("failed to change transfer target file: %w", err)
	}

	transCtx.Transfer.LocalPath = newPath
	transCtx.Transfer.RemotePath = path.Join(
		path.Dir(transCtx.Transfer.RemotePath),
		path.Base(newPath))

	logger.Debug("Changed target file to %q", newPath)

	return nil
}
