package tasks

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// CopyTask is a task which allow to copy the current file
// to a given destination with the same filename.
type CopyTask struct{}

//nolint:gochecknoinits // designed to use init
func init() {
	RunnableTasks["COPY"] = &CopyTask{}
	model.ValidTasks["COPY"] = &CopyTask{}
}

// Validate check if the task has a destination for the copy.
func (*CopyTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy task without a `path` argument: %w", errBadTaskArguments)
	}

	return nil
}

// Run copy the current file to the destination.
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
	err := os.MkdirAll(filepath.Dir(dest), 0o700)
	if err != nil {
		return fmt.Errorf("cannot create destination directory %q: %w",
			filepath.Dir(dest), err)
	}

	srcFile, err := os.Open(filepath.Clean(source))
	if err != nil {
		return normalizeFileError(err)
	}

	//nolint:errcheck // checking errors in deferred function seems useless
	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(dest)
	if err != nil {
		return normalizeFileError(err)
	}

	//nolint:errcheck // checking errors in deferred function seems useless
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("cannot write file to %q: %w", dest, err)
	}

	return nil
}
