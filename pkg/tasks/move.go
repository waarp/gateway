package tasks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// MoveTask is a task which moves the file without renaming it
// the transfer model is modified to reflect this change.
type MoveTask struct{}

func init() {
	RunnableTasks["MOVE"] = &MoveTask{}
	model.ValidTasks["MOVE"] = &MoveTask{}
}

// Warning: both 'oldPath' and 'newPath' must be in denormalized format.
func fallbackMove(dest, source string) error {
	if err := doCopy(dest, source); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		return normalizeFileError(err)
	}

	return nil
}

// MoveFile moves the given file to the given location. Works across partitions.
func MoveFile(source, dest string) error {
	trueSource := utils.DenormalizePath(source)
	trueDest := utils.DenormalizePath(dest)

	if err := os.Rename(trueSource, trueDest); err != nil {
		if _, ok := err.(*os.LinkError); ok {
			return fallbackMove(trueDest, trueSource)
		}
		return normalizeFileError(err)
	}
	return nil
}

// Validate checks if the MOVE task has all the required arguments.
func (*MoveTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move task without a `path` argument")
	}
	return nil
}

// Run executes the task by moving the file in the requested directory.
// TODO Create directory if not exist
func (*MoveTask) Run(args map[string]string, processor *Processor) (string, error) {
	newDir := args["path"]

	source := processor.Transfer.TrueFilepath
	dest := path.Join(newDir, filepath.Base(source))

	if err := MoveFile(source, dest); err != nil {
		return err.Error(), err
	}
	processor.Transfer.TrueFilepath = dest
	return "", nil
}
