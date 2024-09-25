package tasks

import (
	"context"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// copyTask is a task which allow to copy the current file
// to a given destination with the same filename.
type copyTask struct{}

//nolint:gochecknoinits // designed to use init
func init() {
	model.ValidTasks["COPY"] = &copyTask{}
}

// Validate check if the task has a destination for the copy.
func (*copyTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy task without a `path` argument: %w", ErrBadTaskArguments)
	}

	return nil
}

// Run copies the current file to the destination.
func (*copyTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newDir := args["path"]
	source := &transCtx.Transfer.LocalPath

	dest, err := types.ParsePath(newDir)
	if err != nil {
		return fmt.Errorf("failed to parse destination path %q: %w", newDir, err)
	}

	dest = dest.JoinPath(path.Base(source.Path))

	if err := makeCopy(db, transCtx, source, dest); err != nil {
		return err
	}

	logger.Debug("Copied file %q to %q", source, dest)

	return nil
}

func makeCopy(db *database.DB, transCtx *model.TransferContext, source, dest *types.FSPath) error {
	if dest.String() == source.String() {
		// If source == destination, this is a self-copy, so we do nothing.
		return nil
	}

	dstFS, fsErr2 := fs.GetFileSystem(db, dest)
	if fsErr2 != nil {
		return fmt.Errorf("failed to instantiate filesystem for destination %q: %w", dest, fsErr2)
	}

	if err := doCopy(transCtx.FS, dstFS, source, dest); err != nil {
		return err
	}

	transCtx.FS = dstFS

	return nil
}

// doCopy copies the file pointed by the given transfer to the given destination,
// and then returns the filesystem on which the copy was made.
func doCopy(srcFS, dstFS fs.FS, source, dest *types.FSPath) error {
	if err := fs.MkdirAll(srcFS, dest.Dir()); err != nil {
		return fmt.Errorf("cannot create destination directory %q: %w", dest.Dir(), err)
	}

	srcFile, srcErr := fs.Open(srcFS, source)
	if srcErr != nil {
		return fmt.Errorf("failed to open source file %q: %w", source, srcErr)
	}
	defer srcFile.Close() //nolint:errcheck // this should never return an error

	dstFile, dstErr := fs.Create(dstFS, dest)
	if dstErr != nil {
		return fmt.Errorf("failed to create destination file %q: %w", dest, dstErr)
	}
	defer dstFile.Close() //nolint:errcheck // this error is checked elsewhere

	dstFileWriter, canWrite := dstFile.(io.Writer)
	if !canWrite {
		return fmt.Errorf("%w: %s", fs.ErrNotImplemented, "WriteFile")
	}

	if _, err := io.Copy(dstFileWriter, srcFile); err != nil {
		return fmt.Errorf("failed to copy file %q to %q: %w", source, dest, err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("failed to write content to %q: %w", dest, err)
	}

	return nil
}
