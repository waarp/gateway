package tasks

import (
	"fmt"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// MoveTask is a task which moves the file whithout renaming it
// the transfer model is modified to reflect this change.
type MoveTask struct{}

func init() {
	RunnableTasks["MOVE"] = &MoveTask{}
	model.ValidTasks["MOVE"] = &MoveTask{}
}

// Validate checks if the MOVE task has all the required arguments.
func (*MoveTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move task without a `path` argument")
	}
	return nil
}

// Run exects the task by moving the file in the requested directory.
// TODO Create directory if not exist
func (*MoveTask) Run(args map[string]string, processor *Processor) (string, error) {
	var oldPath *string
	newDir := args["path"]

	if processor.Rule.IsSend {
		oldPath = &(processor.Transfer.SourcePath)
	} else {
		oldPath = &(processor.Transfer.DestPath)
	}

	newPath := filepath.Join(newDir, filepath.Base(*oldPath))

	if err := os.Rename(*oldPath, newPath); err != nil {
		return err.Error(), err
	}

	*oldPath = newPath
	return "", nil
}
