package tasks

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// moveTask is a task which moves the file without renaming it
// the transfer model is modified to reflect this change.
type moveTask struct{}

func init() {
	model.ValidTasks["MOVE"] = &moveTask{}
}

// Warning: both 'oldPath' and 'newPath' must be in denormalized format.
func fallbackMove(dest, source string) error {
	if err := doCopy(dest, source); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		return normalizeFileError("delete old file", err)
	}

	return nil
}

// MoveFile moves the given file to the given location. Works across partitions.
func MoveFile(source, dest string) error {
	trueSource := utils.ToOSPath(source)
	trueDest := utils.ToOSPath(dest)

	if _, err := os.Stat(filepath.Dir(trueDest)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(trueDest), 0700); err != nil {
			return normalizeFileError("create target directory", err)
		}
	}

	if err := os.Rename(trueSource, trueDest); err != nil {
		if _, ok := err.(*os.LinkError); ok {
			return fallbackMove(trueDest, trueSource)
		}
		return normalizeFileError("rename file", err)
	}
	return nil
}

// Validate checks if the MOVE task has all the required arguments.
func (*moveTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move task without a `path` argument")
	}
	return nil
}

// Run executes the task by moving the file in the requested directory.
func (*moveTask) Run(args map[string]string, _ *database.DB,
	transCtx *model.TransferContext, _ context.Context) (string, error) {
	newDir := args["path"]

	source := transCtx.Transfer.LocalPath
	dest := path.Join(utils.ToStandardPath(newDir), filepath.Base(source))

	if err := MoveFile(source, dest); err != nil {
		return err.Error(), err
	}
	transCtx.Transfer.LocalPath = utils.ToStandardPath(dest)
	return "", nil
}
