package tasks

import (
	"context"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// deleteTask is a task which delete the current file from the system
type deleteTask struct{}

func init() {
	model.ValidTasks["DELETE"] = &deleteTask{}
}

// Validate the task
func (*deleteTask) Validate(map[string]string) error {
	return nil
}

// Run delete the current file from the system
func (*deleteTask) Run(_ context.Context, _ map[string]string, _ *database.DB, transCtx *model.TransferContext) (string, error) {
	truePath := utils.ToOSPath(transCtx.Transfer.LocalPath)
	if err := os.Remove(truePath); err != nil {
		return "", normalizeFileError("delete file", err)
	}

	return "", nil
}
