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

// copyTask is a task which allow to copy the current file
// to a given destination with the same filename.
type copyTask struct{}

var ErrCopyMissingPath = fmt.Errorf(
	`cannot create a COPY task without a "path" argument: %w`, ErrBadTaskArguments)

// Validate check if the task has a destination for the copy.
func (*copyTask) Validate(args map[string]string) error {
	if args["path"] == "" {
		return ErrCopyMissingPath
	}

	return nil
}

// Run copies the current file to the destination.
func (*copyTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newDir := args["path"]
	if newDir == "" {
		return ErrCopyMissingPath
	}

	source := transCtx.Transfer.LocalPath
	dest := path.Join(newDir, path.Base(source))

	if err := fs.CopyFile(source, dest); err != nil {
		return fmt.Errorf("COPY task failed: %w", err)
	}

	logger.Debug("Copied file %q to %q", source, dest)

	return nil
}
