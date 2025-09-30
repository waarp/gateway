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

type Permissions struct {
	Transfers *Rights
	Servers   *Rights
	Partners  *Rights
	Rules     *Rights
	Users     *Rights
	Admin     *Rights
}

type Rights struct {
	CanRead, CanWrite, CanDelete bool
}

func ParsePermissions(mask model.PermsMask) *Permissions {
	return &Permissions{
		Transfers: &Rights{
			CanRead:   mask.HasPermission(model.PermTransfersRead),
			CanWrite:  mask.HasPermission(model.PermTransfersWrite),
			CanDelete: false,
		},
		Servers: &Rights{
			CanRead:   mask.HasPermission(model.PermServersRead),
			CanWrite:  mask.HasPermission(model.PermServersWrite),
			CanDelete: mask.HasPermission(model.PermServersDelete),
		},
		Partners: &Rights{
			CanRead:   mask.HasPermission(model.PermPartnersRead),
			CanWrite:  mask.HasPermission(model.PermPartnersWrite),
			CanDelete: mask.HasPermission(model.PermPartnersDelete),
		},
		Rules: &Rights{
			CanRead:   mask.HasPermission(model.PermRulesRead),
			CanWrite:  mask.HasPermission(model.PermRulesWrite),
			CanDelete: mask.HasPermission(model.PermRulesDelete),
		},
		Users: &Rights{
			CanRead:   mask.HasPermission(model.PermUsersRead),
			CanWrite:  mask.HasPermission(model.PermUsersWrite),
			CanDelete: mask.HasPermission(model.PermUsersDelete),
		},
		Admin: &Rights{
			CanRead:   mask.HasPermission(model.PermAdminRead),
			CanWrite:  mask.HasPermission(model.PermAdminWrite),
			CanDelete: mask.HasPermission(model.PermAdminDelete),
		},
	}
}
