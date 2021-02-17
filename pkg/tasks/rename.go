package tasks

import (
	"context"
	"fmt"
	"os"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// renameTask is a task which rename the target of the transfer
// without impacting the filesystem.
type renameTask struct{}

func init() {
	model.ValidTasks["RENAME"] = &renameTask{}
}

// Validate checks if the RENAME tasks has all the required arguments.
func (*renameTask) Validate(args map[string]string) error {
	if _, ok := args["path"]; !ok {
		return fmt.Errorf("cannot create a rename task without a `path` argument")
	}
	return nil
}

// Run executes the task by renaming the transfer file
func (*renameTask) Run(args map[string]string, _ *database.DB,
	info *model.TransferContext, _ context.Context) (string, error) {
	newPath := args["path"]

	if _, err := os.Stat(utils.ToOSPath(newPath)); err != nil {
		return "", normalizeFileError("change transfer target file to", err)
	}

	info.Transfer.LocalPath = utils.ToStandardPath(newPath)
	info.Transfer.RemotePath = path.Join(path.Dir(info.Transfer.RemotePath),
		path.Base(info.Transfer.LocalPath))

	return "", nil
}
