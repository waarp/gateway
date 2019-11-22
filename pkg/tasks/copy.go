package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// CopyTask is a task which allow to copy the current file
// to a given destination with the same filename
type CopyTask struct{}

func init() {
	RunnableTasks["COPY"] = &CopyTask{}
}

// Validate check if the task has a destination for the copy
func (*CopyTask) Validate(t *model.Task) error {
	args := map[string]interface{}{}
	if err := json.Unmarshal(t.Args, &args); err != nil {
		return err
	}
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy task without a `path` argument")
	}
	return nil
}

// Run copy the current file to the destination
func (*CopyTask) Run(args map[string]interface{}, runner *Processor) (string, error) {
	newPath := args["path"].(string)
	if runner.Rule.IsSend {
		basename := filepath.Base(runner.Transfer.SourcePath)
		dest := fmt.Sprintf("%s/%s", newPath, basename)

		if err := doCopy(dest, runner.Transfer.SourcePath); err != nil {
			return err.Error(), err
		}
	} else {
		basename := filepath.Base(runner.Transfer.DestPath)
		dest := fmt.Sprintf("%s/%s", newPath, basename)

		if err := doCopy(dest, runner.Transfer.DestPath); err != nil {
			return err.Error(), err
		}
	}
	return "", nil
}

func doCopy(dest, source string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(destFile, srcFile)
	return err
}
