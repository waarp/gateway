package tasks

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
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
			errBadTaskArguments)
	}

	return nil
}

// Run move and rename the current file to the destination and
// modify the transfer model to reflect the file change.
func (*moveRenameTask) Run(_ context.Context, args map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	newPath := args["path"]
	oldPath := transCtx.Transfer.LocalPath

	if err := MoveFile(oldPath, newPath); err != nil {
		return err
	}

	transCtx.Transfer.LocalPath = utils.ToOSPath(newPath)
	transCtx.Transfer.RemotePath = path.Join(path.Dir(transCtx.Transfer.RemotePath),
		filepath.Base(transCtx.Transfer.LocalPath))

	logger.Debug("Moved file %q to %q", oldPath, newPath)

	return nil
}
