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

// moveTask is a task which moves the file without renaming it
// the transfer model is modified to reflect this change.
type moveTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["MOVE"] = &moveTask{}
}

var ErrMoveMissingPath = fmt.Errorf(
	`cannot create a MOVE task without a "path" argument: %w`, ErrBadTaskArguments)

// Validate checks if the MOVE task has all the required arguments.
func (*moveTask) Validate(args map[string]string) error {
	if args["path"] == "" {
		return ErrMoveMissingPath
	}

	return nil
}

// Run executes the task by moving the file in the requested directory.
func (*moveTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newDir := args["path"]
	if newDir == "" {
		return ErrMoveMissingPath
	}

	source := transCtx.Transfer.LocalPath
	dest := path.Join(newDir, path.Base(source))

	if movErr := fs.MoveFile(source, dest); movErr != nil {
		return fmt.Errorf("MOVE task failed: %w", movErr)
	}

	transCtx.Transfer.LocalPath = dest

	logger.Debug("Moved file %q to %q", source, dest)

	return nil
}
