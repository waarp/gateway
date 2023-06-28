package tasks

import (
	"context"
	"errors"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs"
)

// moveTask is a task which moves the file without renaming it
// the transfer model is modified to reflect this change.
type moveTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["MOVE"] = &moveTask{}
}

// Warning: both 'oldPath' and 'newPath' must be in denormalized format.
func fallbackMove(source, dest *types.URL) error {
	if err := doCopy(source, dest); err != nil {
		return err
	}

	if err := fs.Remove(source); err != nil {
		return fmt.Errorf("failed to delete old file %q: %w", source, err)
	}

	return nil
}

// MoveFile moves the given file to the given location. Works across partitions.
func MoveFile(source, dest *types.URL) error {
	// If the source & destination are on different machines, we must use the
	// fallback method of making a copy and then deleting the original.
	if !fs.IsOnSameFS(source, dest) {
		return fallbackMove(source, dest)
	}

	// At this point, we know that the source & dest are on the same filesystem.

	if err := fs.MkdirAll(dest.Dir()); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	if err := fs.Rename(source, dest); err != nil {
		// Despite being on the same file system, if the source & dest are on
		// different partitions, the rename will fail with a LinkError. In this
		// case, we must fall back to a copy (like when the files were on
		// different machines).
		if errors.As(err, new(*fs.LinkError)) {
			return fallbackMove(source, dest)
		}

		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// Validate checks if the MOVE task has all the required arguments.
func (*moveTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move task without a `path` argument: %w",
			errBadTaskArguments)
	}

	return nil
}

// Run executes the task by moving the file in the requested directory.
func (*moveTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newDir := args["path"]
	source := &transCtx.Transfer.LocalPath

	dirPath, dstErr := types.ParseURL(newDir)
	if dstErr != nil {
		return fmt.Errorf("failed to parse the MOVE destination path %q: %w", newDir, dstErr)
	}

	dest := dirPath.JoinPath(path.Base(source.Path))

	if err := MoveFile(source, dest); err != nil {
		return err
	}

	transCtx.Transfer.LocalPath = *dest

	logger.Debug("Moved file %q to %q", source, dest)

	return nil
}
