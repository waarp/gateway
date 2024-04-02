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

// renameTask is a task which rename the target of the transfer
// without impacting the filesystem.
type renameTask struct{}

//nolint:gochecknoinits // init is used by design
func init() {
	model.ValidTasks["RENAME"] = &renameTask{}
}

// Validate checks if the RENAME tasks has all the required arguments.
func (*renameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a rename task without a `path` argument: %w", ErrBadTaskArguments)
	}

	return nil
}

// Run executes the task by renaming the transfer file.
func (*renameTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newPath := args["path"]

	newURL, err := types.ParseURL(newPath)
	if err != nil {
		return fmt.Errorf("failed to parse the new target path %q: %w", newPath, err)
	}

	newFS, fsErr := fs.GetFileSystem(db, newURL)
	if fsErr != nil {
		return fmt.Errorf("failed to instantiate filesystem for new file %q: %w", newPath, fsErr)
	}

	if _, err := fs.Stat(transCtx.FS, newURL); err != nil {
		return fmt.Errorf("failed to change transfer target file: %w", err)
	}

	transCtx.FS = newFS
	transCtx.Transfer.LocalPath = *newURL
	transCtx.Transfer.RemotePath = path.Join(path.Dir(transCtx.Transfer.RemotePath),
		path.Base(newURL.Path))

	logger.Debug("Changed target file to %q", newPath)

	return nil
}
