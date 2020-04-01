package tasks

import (

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
	model.ValidTasks["COPY"] = &CopyTask{}
}

// Validate check if the task has a destination for the copy
func (*CopyTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy task without a `path` argument")
	}
	return nil
}

// Run copy the current file to the destination
func (*CopyTask) Run(args map[string]string, runner *Processor) (string, error) {
	var (
		newDir  = args["path"]
		oldPath string
	)

	if runner.Rule.IsSend {
		oldPath = runner.Transfer.SourcePath
	} else {
		oldPath = runner.Transfer.DestPath
	}

	newPath := filepath.Join(newDir, filepath.Base(oldPath))

	if err := doCopy(newPath, oldPath); err != nil {
		return err.Error(), err
	}

	return "", nil
}

func doCopy(dest, source string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
