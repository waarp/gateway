package common

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetUser(r *http.Request) *model.User {
	//nolint:forcetypeassert //assertion always succeeds
	return r.Context().Value(constants.ContextUserKey).(*model.User)
}
