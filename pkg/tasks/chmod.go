package tasks

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type ChmodRemotePathError string

func (e ChmodRemotePathError) Error() string {
	return fmt.Sprintf("%q is not a local path", string(e))
}

type ChmodBitsError fs.FileMode

func (e ChmodBitsError) Error() string {
	return fmt.Sprintf(`invalid file permissions "%o"`, e)
}

type chmodTask struct {
	Perms string `json:"perms"`

	mode fs.FileMode
}

func (c *chmodTask) Validate(args map[string]string) error {
	if err := utils.JSONConvert(args, c); err != nil {
		return fmt.Errorf("failed to parse chmod parameters: %w", err)
	}

	perms, err := strconv.ParseUint(c.Perms, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid file permissions %q: %w", c.Perms, err)
	}

	c.mode = fs.FileMode(perms)
	if c.mode == 0 || c.mode&fs.ModePerm != c.mode {
		return ChmodBitsError(c.mode)
	}

	return nil
}

func (c *chmodTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := c.Validate(params); err != nil {
		return err
	}

	path := transCtx.Transfer.LocalPath
	if !fs.IsLocalPath(path) {
		return ChmodRemotePathError(path)
	}

	if err := os.Chmod(path, c.mode); err != nil {
		return fmt.Errorf("failed to change file permissions: %w", err)
	}

	logger.Debugf("Changed file permissions to %03o", c.mode)

	return nil
}
