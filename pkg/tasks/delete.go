package tasks

import (
	"os"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// DeleteTask is a task which delete the current file from the system.
type DeleteTask struct{}

//nolint:gochecknoinits // designed to use init
func init() {
	RunnableTasks["DELETE"] = &DeleteTask{}
	model.ValidTasks["DELETE"] = &DeleteTask{}
}

// Validate the task.
func (*DeleteTask) Validate(map[string]string) error {
	return nil
}

// Run delete the current file from the system.
func (*DeleteTask) Run(_ map[string]string, runner *Processor) (string, error) {
	truePath := utils.DenormalizePath(runner.Transfer.TrueFilepath)

	if err := os.Remove(truePath); err != nil {
		err2 := normalizeFileError(err)

		return err2.Error(), err2
	}

	runner.Transfer.TrueFilepath = ""

	return "", nil
}
