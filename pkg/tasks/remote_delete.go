package tasks

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrRemoteDeleteNotSupported = errors.New("the protocol or agent does not support remote delete")

type remoteDelete struct {
	Path      string       `json:"path"`
	Timeout   jsonDuration `json:"timeout"`
	Recursive jsonBool     `json:"recursive"`
}

func (r *remoteDelete) parseArgs(args map[string]string) error {
	*r = remoteDelete{}

	if err := utils.JSONConvert(args, r); err != nil {
		return fmt.Errorf("failed to parse remote delete task arguments: %w", err)
	}

	return nil
}

func (r *remoteDelete) Validate(args map[string]string) error {
	return r.parseArgs(args)
}

func (r *remoteDelete) Run(ctx context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, remote any,
) error {
	deleter, ok := remote.(RemoteDeleter)
	if !ok {
		return ErrRemoteDeleteNotSupported
	}

	if err := r.parseArgs(params); err != nil {
		return err
	}

	file := transCtx.Transfer.RemotePath
	if r.Path != "" {
		file = r.Path
	}

	if !r.Timeout.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout.Duration)
		defer cancel()
	}

	if err := deleter.Delete(ctx, file, bool(r.Recursive)); err != nil {
		return fmt.Errorf("failed to delete remote file %q: %w", file, err)
	}

	logger.Debugf("Deleted remote file %q", file)

	return nil
}
