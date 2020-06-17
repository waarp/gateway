package tasks

import (
	"fmt"
	"io"
	"os"
	"path"
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
	newDir := args["path"]

	source := runner.Transfer.TrueFilepath
	dest := path.Join(newDir, filepath.Base(source))

	if err := doCopy(dest, source); err != nil {
		return err.Error(), err
	}

	return "", nil
}

func doCopy(dest, source string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return normalizeFileError(err)
	}
	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(dest)
	if err != nil {
		return normalizeFileError(err)
	}
	defer func() { _ = destFile.Close() }()
	_, err = io.Copy(destFile, srcFile)
	return err
}
