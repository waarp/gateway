package tasks

import (
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// DeleteTask is a task which delete the current file from the system
type DeleteTask struct{}

func init() {
	RunnableTasks["DELETE"] = &DeleteTask{}
	model.ValidTasks["DELETE"] = &DeleteTask{}
}

// Validate the task
func (*DeleteTask) Validate(map[string]string) error {
	return nil
}

// Run delete the current file from the system
func (*DeleteTask) Run(args map[string]interface{}, runner *Processor) (string, error) {
	if runner.Rule.IsSend {
		if err := os.Remove(filepath.FromSlash(runner.Transfer.SourcePath)); err != nil {
			return err.Error(), err
		}
	} else {
		if err := os.Remove(runner.Transfer.DestPath); err != nil {
			return err.Error(), err
		}
	}
	return "", nil
}
