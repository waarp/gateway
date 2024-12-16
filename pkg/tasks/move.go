package tasks

import (
	"context"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// moveTask is a task which moves the file without renaming it
// the transfer model is modified to reflect this change.
type moveTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["MOVE"] = &moveTask{}
}

// Warning: both 'oldPath' and 'newPath' must be in denormalized format.
func fallbackMove(srcFS, dstFS fs.FS, source, dest *types.FSPath) error {
	if err := doCopy(srcFS, dstFS, source, dest); err != nil {
		return err
	}

	if err := fs.Remove(srcFS, source); err != nil {
		return fmt.Errorf("failed to delete old file %q: %w", source, err)
	}

	return nil
}

// MoveFile moves the given file to the given location. Works across partitions.
func MoveFile(db database.ReadAccess, srcFS fs.FS, source, dest *types.FSPath) (fs.FS, error) {
	// If the source & destination are on different machines, we must use the
	// fallback method of making a copy and then deleting the original.
	if !fs.IsOnSameFS(source, dest) {
		dstFS, fsErr2 := fs.GetFileSystem(db, dest)
		if fsErr2 != nil {
			return nil, fmt.Errorf("failed to instantiate filesystem for destination %q: %w", dest, fsErr2)
		}

		if err := fallbackMove(srcFS, dstFS, source, dest); err != nil {
			return nil, err
		}

		return dstFS, nil
	}

	// At this point, we know that the source & dest are on the same filesystem.
	if err := fs.MkdirAll(srcFS, dest.Dir()); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	if err := fs.Rename(srcFS, source, dest); err != nil {
		// Source & dest might be on the same FS but on different partitions.
		// So if Rename fails, we try the fallback method instead.
		if err2 := fallbackMove(srcFS, srcFS, source, dest); err2 == nil {
			return srcFS, nil
		}

		return nil, fmt.Errorf("failed to rename file: %w", err)
	}

	return srcFS, nil
}

// Validate checks if the MOVE task has all the required arguments.
func (*moveTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move task without a `path` argument: %w",
			ErrBadTaskArguments)
	}

	return nil
}

// Run executes the task by moving the file in the requested directory.
func (*moveTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newDir := args["path"]
	source := &transCtx.Transfer.LocalPath

	dirPath, dstErr := types.ParsePath(newDir)
	if dstErr != nil {
		return fmt.Errorf("failed to parse the MOVE destination path %q: %w", newDir, dstErr)
	}

	dest := dirPath.JoinPath(path.Base(source.Path))

	newFS, movErr := MoveFile(db, transCtx.FS, source, dest)
	if movErr != nil {
		return movErr
	}

	transCtx.FS = newFS
	transCtx.Transfer.LocalPath = *dest

	logger.Debug("Moved file %q to %q", source, dest)

	return nil
}
