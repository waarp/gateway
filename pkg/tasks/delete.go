package tasks

import (
	"fmt"
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
func (*DeleteTask) Run(_ map[string]string, runner *Processor) (string, error) {
	fmt.Printf("\n%s\n", runner.Transfer.TrueFilepath)
	if err := os.Remove(filepath.FromSlash(runner.Transfer.TrueFilepath)); err != nil {
		return err.Error(), err
	}
	runner.Transfer.TrueFilepath = ""
	return "", nil
}
