package tasks

import (
	"context"
	"fmt"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// moveRenameTask is a task which move and rename the current file
// to a given destination
type moveRenameTask struct{}

func init() {
	model.ValidTasks["MOVERENAME"] = &moveRenameTask{}
}

// Validate check if the task has a destination for the move.
func (*moveRenameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a move_rename task without a `path` argument")
	}
	return nil
}

// Run move and rename the current file to the destination and
// modify the transfer model to reflect the file change.
func (*moveRenameTask) Run(args map[string]string, _ *database.DB,
	info *model.TransferContext, _ context.Context) (string, error) {
	newPath := args["path"]
	oldPath := info.Transfer.TrueFilepath

	if err := MoveFile(oldPath, newPath); err != nil {
		return err.Error(), err
	}
	info.Transfer.TrueFilepath = utils.NormalizePath(newPath)
	if info.Rule.IsSend {
		info.Transfer.SourceFile = path.Base(info.Transfer.TrueFilepath)
	} else {
		info.Transfer.DestFile = path.Base(info.Transfer.TrueFilepath)
	}
	return "", nil
}
