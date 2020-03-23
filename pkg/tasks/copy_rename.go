package tasks

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// CopyRenameTask is a task which allow to copy the current file
// to a given destination
type CopyRenameTask struct{}

func init() {
	RunnableTasks["COPYRENAME"] = &CopyRenameTask{}
	model.ValidTasks["COPYRENAME"] = &CopyRenameTask{}
}

// Validate check if the task has a destination for the copy
func (*CopyRenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy_rename task without a `path` argument")
	}
	return nil
}

// Run copy the current file to the destination
func (*CopyRenameTask) Run(args map[string]string, processor *Processor) (string, error) {
	var (
		newPath = args["path"]
		srcPath string
	)

	if processor.Rule.IsSend {
		srcPath = processor.Transfer.SourceFile
	} else {
		srcPath = processor.Transfer.DestFile
	}

	if err := doCopy(newPath, srcPath); err != nil {
		return err.Error(), err
	}

	return "", nil
}
