package tasks

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
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

	if _, err := os.Stat(utils.ToOSPath(newPath)); err != nil {
		return normalizeFileError("change transfer target file to", err)
	}

	transCtx.Transfer.LocalPath = utils.ToOSPath(newPath)
	transCtx.Transfer.RemotePath = path.Join(path.Dir(transCtx.Transfer.RemotePath),
		filepath.Base(transCtx.Transfer.LocalPath))

	logger.Debug("Changed target file to %q", newPath)

	return nil
}
