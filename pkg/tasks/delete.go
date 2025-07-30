package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// deleteTask is a task which delete the current file from the system.
type deleteTask struct{}

// Validate the task.
func (*deleteTask) Validate(map[string]string) error {
	return nil
}

// Run deletes the current file from the system.
func (*deleteTask) Run(_ context.Context, _ map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	filepath := transCtx.Transfer.LocalPath

	if err := fs.Remove(filepath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.Debugf("Deleted file %q", filepath)

	return nil
}
