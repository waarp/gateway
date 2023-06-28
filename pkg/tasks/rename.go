package tasks

import (
	"context"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs"
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
		return fmt.Errorf("cannot create a rename task without a `path` argument: %w", errBadTaskArguments)
	}

	return nil
}

// Run executes the task by renaming the transfer file.
func (*renameTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	newPath := args["path"]

	newURL, err := types.ParseURL(newPath)
	if err != nil {
		return fmt.Errorf("failed to parse the new target path %q: %w", newPath, err)
	}

	if _, err := fs.Stat(newURL); err != nil {
		return fmt.Errorf("failed to change transfer target file: %w", err)
	}

	transCtx.Transfer.LocalPath = *newURL
	transCtx.Transfer.RemotePath = path.Join(path.Dir(transCtx.Transfer.RemotePath),
		path.Base(newURL.Path))

	logger.Debug("Changed target file to %q", newPath)

	return nil
}
