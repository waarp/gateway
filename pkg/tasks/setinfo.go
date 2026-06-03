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

var ErrSetInfoMissingKey = errors.New(`missing "key" argument`)

type setInfoTask struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (t *setInfoTask) Validate(args map[string]string) error {
	*t = setInfoTask{}

	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse SETINFO arguments: %w", err)
	}

	if t.Key == "" {
		return ErrSetInfoMissingKey
	}

	return nil
}

func (t *setInfoTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := utils.JSONConvert(args, t); err != nil {
		return fmt.Errorf("failed to parse SETINFO arguments: %w", err)
	}

	if t.Key == "" {
		return ErrSetInfoMissingKey
	}

	if transCtx.Transfer.TransferInfo == nil {
		transCtx.Transfer.TransferInfo = make(map[string]any)
	}

	old, existed := transCtx.Transfer.TransferInfo[t.Key]

	if t.Value == "" {
		// Empty value = delete the key.
		delete(transCtx.Transfer.TransferInfo, t.Key)

		if existed {
			logger.Debugf("SETINFO: deleted key %q (was %v)", t.Key, old)
		}
	} else {
		transCtx.Transfer.TransferInfo[t.Key] = t.Value

		if existed {
			logger.Debugf("SETINFO: updated key %q: %v -> %q", t.Key, old, t.Value)
		} else {
			logger.Debugf("SETINFO: set key %q = %q", t.Key, t.Value)
		}
	}

	return nil
}
