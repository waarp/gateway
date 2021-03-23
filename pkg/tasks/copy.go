package tasks

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// copyTask is a task which allow to copy the current file
// to a given destination with the same filename
type copyTask struct{}

func init() {
	model.ValidTasks["COPY"] = &copyTask{}
}

// Validate check if the task has a destination for the copy
func (*copyTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy task without a `path` argument")
	}
	return nil
}

// Run copy the current file to the destination
func (*copyTask) Run(args map[string]string, _ *database.DB,
	transCtx *model.TransferContext, _ context.Context) (string, error) {
	newDir := args["path"]

	source := transCtx.Transfer.LocalPath
	dest := path.Join(newDir, filepath.Base(source))

	if err := doCopy(dest, source); err != nil {
		return err.Error(), err
	}

	return "", nil
}

func doCopy(dest, source string) error {
	trueSource := utils.ToOSPath(source)
	trueDest := utils.ToOSPath(dest)

	err := os.MkdirAll(filepath.Dir(trueDest), 0o700)
	if err != nil {
		return err
	}

	srcFile, err := os.Open(trueSource)
	if err != nil {
		return normalizeFileError("open source file", err)
	}
	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(trueDest)
	if err != nil {
		return normalizeFileError("create destination file", err)
	}
	defer func() { _ = destFile.Close() }()
	_, err = io.Copy(destFile, srcFile)
	return err
}
