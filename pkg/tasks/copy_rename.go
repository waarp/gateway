package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// copyRenameTask is a task which allow to copy the current file
// to a given destination.
type copyRenameTask struct{}

var ErrCopyRenameMissingPath = fmt.Errorf(
	`cannot create a COPYRENAME task without a "path" argument: %w`, ErrBadTaskArguments)

// Validate check if the task has a destination for the copy.
func (*copyRenameTask) Validate(args map[string]string) error {
	if args["path"] == "" {
		return ErrCopyRenameMissingPath
	}

	return nil
}

// Run copies the current file to the destination.
func (*copyRenameTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	source := transCtx.Transfer.LocalPath
	dest := args["path"]

	if dest == "" {
		return ErrCopyRenameMissingPath
	}

	if err := fs.CopyFile(source, dest); err != nil {
		return fmt.Errorf("COPYRENAME task failed: %w", err)
	}

	logger.Debugf("Copied file %q to %q", source, dest)

	return nil
}
