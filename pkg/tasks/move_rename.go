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

// moveRenameTask is a task which move and rename the current file to a given
// destination.
type moveRenameTask struct{}

var ErrMoveRenameMissingPath = fmt.Errorf(
	`cannot create a MOVERENAME task without a "path" argument: %w`, ErrBadTaskArguments)

// Validate check if the task has a destination for the move.
func (*moveRenameTask) Validate(args map[string]string) error {
	if args["path"] == "" {
		return ErrMoveRenameMissingPath
	}

	return nil
}

// Run move and rename the current file to the destination and
// modify the transfer model to reflect the file change.
func (*moveRenameTask) Run(_ context.Context, args map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	source := transCtx.Transfer.LocalPath
	dest := args["path"]

	if dest == "" {
		return ErrMoveRenameMissingPath
	}

	if movErr := fs.MoveFile(source, dest); movErr != nil {
		return fmt.Errorf("MOVERENAME task failed: %w", movErr)
	}

	transCtx.Transfer.LocalPath = dest
	transCtx.Transfer.RemotePath = path.Join(
		path.Dir(transCtx.Transfer.RemotePath),
		path.Base(transCtx.Transfer.LocalPath))

	logger.Debugf("Moved file %q to %q", source, dest)

	return nil
}
