package tasks

import (
	"fmt"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// MoveRenameTask is a task which move and rename the current file
// to a given destination
type MoveRenameTask struct{}

func init() {
	RunnableTasks["MOVERENAME"] = &MoveRenameTask{}
	model.ValidTasks["MOVERENAME"] = &MoveRenameTask{}
}

// Validate check if the task has a destination for the move.
func (*MoveRenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move_rename task without a `path` argument")
	}
	return nil
}

// Run move and rename the current file to the destination and
// modify the transfer model to reflect the file change.
func (*MoveRenameTask) Run(args map[string]string, processor *Processor) (string, error) {
	newPath := args["path"]
	var oldPath *string

	if processor.Rule.IsSend {
		oldPath = &(processor.Transfer.SourcePath)
	} else {
		oldPath = &(processor.Transfer.DestPath)
	}

	if err := os.Rename(*oldPath, newPath); err != nil {
		return err.Error(), err
	}
	*oldPath = newPath
	return "", nil
}
