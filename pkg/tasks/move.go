package tasks

import (
	"encoding/json"
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
}

// Validate checks if the MOVE task has all the required arguments.
func (*MoveTask) Validate(t *model.Task) error {
	args := map[string]interface{}{}
	if err := json.Unmarshal(t.Args, &args); err != nil {
		return err
	}
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move task without a `path` argument")
	}
	return nil
}

// Run exects the task by moving the file in the requested directory.
func (*MoveTask) Run(args map[string]interface{}, processor *Processor) (string, error) {
	newPath := args["path"].(string)
	if processor.Rule.IsSend {
		name := filepath.Base(processor.Transfer.SourcePath)
		dest := fmt.Sprintf("%s/%s", newPath, name)

		if err := os.Rename(processor.Transfer.SourcePath, dest); err != nil {
			return err.Error(), err
		}
		processor.Transfer.SourcePath = dest
		return "", nil
	}
	name := filepath.Base(processor.Transfer.DestPath)
	dest := fmt.Sprintf("%s/%s", newPath, name)

	if err := os.Rename(processor.Transfer.DestPath, dest); err != nil {
		return err.Error(), err
	}
	processor.Transfer.DestPath = dest
	return "", nil
}
