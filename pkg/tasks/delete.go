package tasks

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	filepath := transCtx.Transfer.LocalPath

	if err := fs.Remove(filepath); errors.Is(err, fs.ErrNotExist) {
		logger.Warningf("Failed to delete file %q: file does not exist", filepath)

		return newWarningf("failed to delete file %q: file does not exist", filepath)
	} else if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.Debugf("Deleted file %q", filepath)

	return nil
}
