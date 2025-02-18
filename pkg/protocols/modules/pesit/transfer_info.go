package pesit

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const (
	clientConnFreetextKey  = "__clientConnFreetext__"
	clientTransFreetextKey = "__clientTransFreetext__"
	serverConnFreetextKey  = "__serverConnFreetext__"
	serverTransFreetextKey = "__serverTransFreetext__"
)

type freetextSetter interface {
	SetFreeText(freetext string) bool
}

func setFreetext(pip *pipeline.Pipeline, key string, f freetextSetter) *pipeline.Error {
	valAny, hasKey := pip.TransCtx.TransInfo[key]
	if !hasKey {
		return nil
	}

	if val, isStr := valAny.(string); isStr {
		f.SetFreeText(val)

		return nil
	}

	return pipeline.NewError(types.TeInternal,
		"freetext variable %q must be a string (was of type %T)", key, valAny)
}

func getFreetext(pip *pipeline.Pipeline, key, freetext string) {
	pip.TransCtx.TransInfo[key] = freetext
}
