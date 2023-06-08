package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// copyRenameTask is a task which allow to copy the current file
// to a given destination.
type copyRenameTask struct{}

//nolint:gochecknoinits // designed to use init
func init() {
	model.ValidTasks["COPYRENAME"] = &copyRenameTask{}
}

// Validate check if the task has a destination for the copy.
func (*copyRenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy_rename task without a `path` argument: %w", errBadTaskArguments)
	}

	return nil
}

// Run copies the current file to the destination.
func (*copyRenameTask) Run(_ context.Context, args map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	dstPath := args["path"]
	srcPath := transCtx.Transfer.LocalPath

	if err := doCopy(dstPath, srcPath); err != nil {
		return err
	}

	logger.Debug("Copied file %q to %q", srcPath, dstPath)

	return nil
}
