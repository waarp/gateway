package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// copyRenameTask is a task which allow to copy the current file
// to a given destination
type copyRenameTask struct{}

func init() {
	model.ValidTasks["COPYRENAME"] = &copyRenameTask{}
}

// Validate check if the task has a destination for the copy
func (*copyRenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a copy_rename task without a `path` argument")
	}
	return nil
}

// Run copy the current file to the destination
func (*copyRenameTask) Run(args map[string]string, _ *database.DB,
	info *model.TransferContext, _ context.Context) (string, error) {

	dstPath := args["path"]
	srcPath := info.Transfer.LocalPath

	if err := doCopy(dstPath, srcPath); err != nil {
		return err.Error(), err
	}

	return "", nil
}
