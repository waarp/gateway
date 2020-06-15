package tasks

import (
	"fmt"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// RenameTask is a task which rename the target of the transfer
// without impacting the filesystem.
type RenameTask struct{}

func init() {
	RunnableTasks["RENAME"] = &RenameTask{}
	model.ValidTasks["RENAME"] = &RenameTask{}
}

// Validate checks if the RENAME tasks has all the required arguments.
func (*RenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a rename task without a `path` argument")
	}
	return nil
}

// Run executes the task by renaming the transfer file
func (*RenameTask) Run(args map[string]string, processor *Processor) (string, error) {
	newPath := args["path"]
	processor.Transfer.TrueFilepath = newPath
	if processor.Rule.IsSend {
		processor.Transfer.SourceFile = path.Base(newPath)
		return "", nil
	}
	processor.Transfer.DestFile = path.Base(newPath)
	return "", nil
}
