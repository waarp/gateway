package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type setInfoTask struct {
	args map[string]jsonValue
}

func (t *setInfoTask) Validate(args map[string]string) error {
	*t = setInfoTask{}

	if err := utils.JSONConvert(args, &t.args); err != nil {
		return fmt.Errorf("failed to parse SETINFO arguments: %w", err)
	}

	return nil
}

func (t *setInfoTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.Validate(args); err != nil {
		return err
	}

	if transCtx.Transfer.TransferInfo == nil {
		transCtx.Transfer.TransferInfo = map[string]any{}
	}

	// Old behavior: set 1 key
	if key, hasKey := args["key"]; hasKey {
		t.setInfo(logger, transCtx, key, t.args["value"])

		return nil
	}

	// New behavior: set multiple keys
	for key, val := range t.args {
		t.setInfo(logger, transCtx, key, val)
	}

	return nil
}

func (t *setInfoTask) setInfo(logger *log.Logger, transCtx *model.TransferContext,
	key string, jv jsonValue,
) {
	old, existed := transCtx.Transfer.TransferInfo[key]
	val := jv.Val

	if val != nil && val != "" {
		transCtx.Transfer.TransferInfo[key] = val

		if existed {
			logger.Debugf("SETINFO: updated key %q: %v -> %v", key, old, val)
		} else {
			logger.Debugf("SETINFO: added key %q = %v", key, val)
		}
	} else if existed {
		// Empty value = delete the key.
		delete(transCtx.Transfer.TransferInfo, key)

		logger.Debugf("SETINFO: deleted key %q (was %v)", key, old)
	}
}
