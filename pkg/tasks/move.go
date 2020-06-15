package tasks

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// MoveTask is a task which moves the file whithout renaming it
// the transfer model is modified to reflect this change.
type MoveTask struct{}

func init() {
	RunnableTasks["MOVE"] = &MoveTask{}
	model.ValidTasks["MOVE"] = &MoveTask{}
}

func fallbackMove(oldPath, newPath string) error {
	src, err := os.Open(oldPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(newPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	if err = os.Remove(oldPath); err != nil {
		return err
	}

	return nil
}

// MoveFile moves the given file to the given location. Works across partitions.
func MoveFile(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		linkErr, ok := err.(*os.LinkError)
		if ok && linkErr.Err.Error() == "invalid cross-device link" {
			return fallbackMove(oldPath, newPath)
		}
		return err
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

// Run exects the task by moving the file in the requested directory.
// TODO Create directory if not exist
func (*MoveTask) Run(args map[string]string, processor *Processor) (string, error) {
	var oldPath *string
	newDir := args["path"]

	oldPath = &(processor.Transfer.TrueFilepath)

	newPath := utils.SlashJoin(newDir, filepath.Base(*oldPath))

	if err := MoveFile(*oldPath, newPath); err != nil {
		return err.Error(), err
	}
	*oldPath = newPath
	return "", nil
}
