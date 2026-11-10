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

// The arguments of the deprecated form of the SETINFO task (before 0.17.0),
// {"key": name, "value": value}, which sets the one key it names. Both names
// stay reserved to that form: no transfer information can be named after them.
const (
	setInfoKeyArg   = "key"
	setInfoValueArg = "value"
)

// ErrSetInfoReservedArgs is returned when the arguments of a SETINFO task use
// "key" or "value" otherwise than as its deprecated form alone.
var ErrSetInfoReservedArgs = errors.New(`the "key" and "value" arguments of SETINFO are reserved to its ` +
	`deprecated form (the one key named by "key", set to "value") and cannot be mixed with other keys`)

type setInfoTask struct {
	args map[string]jsonValue
}

// SetInfoPairs are the transfer information a SETINFO task sets, read from its
// arguments: each argument names a key and gives its value, except in the
// deprecated form {"key": name, "value": value}, which sets the one key it
// names (and removes it when "value" is missing). The other arguments of a
// mixed form are dropped, as the task drops them when it runs; Validate
// refuses such a form.
func SetInfoPairs(args map[string]string) map[string]string {
	if key, deprecated := args[setInfoKeyArg]; deprecated {
		return map[string]string{key: args[setInfoValueArg]}
	}

	return args
}

// SetInfoReserved tells whether a name is one of the arguments reserved to the
// deprecated form of SETINFO, which no transfer information can be named after.
func SetInfoReserved(name string) bool {
	return name == setInfoKeyArg || name == setInfoValueArg
}

// Validate parses the arguments, in either form, and refuses "key" and "value"
// used otherwise than as the deprecated form alone: such a task would set the
// one key of that form and drop the others without a word.
func (t *setInfoTask) Validate(args map[string]string) error {
	if err := t.parse(args); err != nil {
		return err
	}

	// In the deprecated form, every argument is a reserved one; in the current
	// form, none is.
	_, deprecated := args[setInfoKeyArg]

	for name := range args {
		if SetInfoReserved(name) != deprecated {
			return ErrSetInfoReservedArgs
		}
	}

	return nil
}

// parse reads the arguments, in either form, as the keys to set and their
// typed values.
func (t *setInfoTask) parse(args map[string]string) error {
	*t = setInfoTask{}

	if err := utils.JSONConvert(SetInfoPairs(args), &t.args); err != nil {
		return fmt.Errorf("failed to parse SETINFO arguments: %w", err)
	}

	return nil
}

func (t *setInfoTask) Run(_ context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	// Parsed, not validated: a stored task of a mixed form keeps running as it
	// did, setting the one key of its deprecated form.
	if err := t.parse(args); err != nil {
		return err
	}

	if transCtx.Transfer.TransferInfo == nil {
		transCtx.Transfer.TransferInfo = map[string]any{}
	}

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
