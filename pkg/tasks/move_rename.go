package tasks

import (
	"context"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// moveRenameTask is a task which move and rename the current file to a given
// destination.
type moveRenameTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["MOVERENAME"] = &moveRenameTask{}
}

// Validate check if the task has a destination for the move.
func (*moveRenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move_rename task without a `path` argument: %w",
			ErrBadTaskArguments)
	}

	return nil
}

// Run move and rename the current file to the destination and
// modify the transfer model to reflect the file change.
func (*moveRenameTask) Run(_ context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	newPath := args["path"]
	source := &transCtx.Transfer.LocalPath

	dest, dstErr := types.ParseURL(newPath)
	if dstErr != nil {
		return fmt.Errorf("failed to parse the MOVE destination path %q: %w", newPath, dstErr)
	}

	newFS, movErr := MoveFile(db, transCtx.FS, source, dest)
	if movErr != nil {
		return movErr
	}

	transCtx.FS = newFS
	transCtx.Transfer.LocalPath = *dest
	transCtx.Transfer.RemotePath = path.Join(
		path.Dir(transCtx.Transfer.RemotePath),
		path.Base(transCtx.Transfer.LocalPath.Path))

	logger.Debug("Moved file %q to %q", source, newPath)

	return nil
}
