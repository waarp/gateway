//nolint:dupl // in process
package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type userPermissions struct {
	Username    string
	Permissions *model.Permissions
}

func listUser(db *database.DB, r *http.Request) []*model.User {
	orderAsc := false
	limit := 0
	var err error

	orderAscRes := r.URL.Query().Get("orderAsc")
	if orderAscRes == "true" {
		orderAsc = true
	}

	limitRes := r.URL.Query().Get("limit")
	if limitRes != "" {
		limit, err = strconv.Atoi(limitRes)
		if err != nil {
			limit = 0
		}
	}

	user, err := internal.ListUsers(db, "username", orderAsc, limit, 0)
	if err != nil {
		return nil
	}

	return user
}

func userManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("user_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		userList := listUser(db, r)

		var uPermissionsList []userPermissions
		for _, u := range userList {
			uPermissionsList = append(uPermissionsList, userPermissions{
				Username:    u.Username,
				Permissions: model.MaskToPerms(u.Permissions),
			})
		}

		user, err := GetUserByToken(r, db)
		myPermission := model.MaskToPerms(user.Permissions)

		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		if err := userManagementTemplate.ExecuteTemplate(w, "user_management_page", map[string]any{
			"userPermissions": uPermissionsList,
			"myPermission":    myPermission,
			"tab":             tabTranslated,
			"username":        user.Username,
			"language":        userLanguage,
		}); err != nil {
			logger.Error("render user_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
