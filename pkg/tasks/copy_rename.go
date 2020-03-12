package tasks

import (
	"encoding/json"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// CopyRenameTask is a task which allow to copy the current file
// to a given destination
type CopyRenameTask struct{}

func init() {
	RunnableTasks["COPYRENAME"] = &CopyRenameTask{}
}

// Validate check if the task has a destination for the copy
func (*CopyRenameTask) Validate(t *model.Task) error {
	args := map[string]string{}
	if err := json.Unmarshal(t.Args, &args); err != nil {
		return err
	}
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
		srcPath = processor.Transfer.SourcePath
	} else {
		srcPath = processor.Transfer.DestPath
	}

	if err := doCopy(newPath, srcPath); err != nil {
		return err.Error(), err
	}

	return "", nil
}
