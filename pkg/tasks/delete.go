package tasks

import (
	"context"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// deleteTask is a task which delete the current file from the system.
type deleteTask struct{}

//nolint:gochecknoinits // designed to use init
func init() {
	model.ValidTasks["DELETE"] = &deleteTask{}
}

// Validate the task.
func (*deleteTask) Validate(map[string]string) error {
	return nil
}

// Run deletes the current file from the system.
func (*deleteTask) Run(_ context.Context, _ map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	truePath := utils.ToOSPath(transCtx.Transfer.LocalPath)
	if err := os.Remove(truePath); err != nil {
		return normalizeFileError("delete file", err)
	}

	logger.Debug("Deleted file %q", truePath)

	return nil
}
