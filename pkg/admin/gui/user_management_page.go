//nolint:dupl // in process
package gui

import (
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type userPermissions struct {
	Username    string
	ID          int64
	Permissions *model.Permissions
}

func addUser(db *database.DB, r *http.Request) error {
	var newUser model.User
	var permissions model.Permissions
	var err error

	newUserUsername := r.URL.Query().Get("newUserUsername")
	if newUserUsername != "" {
		newUser.Username = newUserUsername
	}

	newUserPassword := r.URL.Query().Get("newUserPassword")
	if newUserPassword != "" {
		newUser.PasswordHash, err = internal.HashPassword(newUserPassword)
		if err != nil {
			return fmt.Errorf("internal error: %w", err)
		}
	}
	permissions.Transfers = r.URL.Query().Get("newUserPermissionsTransfers")
	permissions.Servers = r.URL.Query().Get("newUserPermissionServers")
	permissions.Partners = r.URL.Query().Get("newUserPermissionsPartners")
	permissions.Rules = r.URL.Query().Get("newUserPermissionsRules")
	permissions.Users = r.URL.Query().Get("newUserPermissionsUsers")
	permissions.Administration = r.URL.Query().Get("newUserPermissionsAdministration")

	newUser.Permissions, err = model.PermsToMask(&permissions)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.InsertUser(db, &newUser); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func editUser(db *database.DB, r *http.Request) error {
	var permissions model.Permissions
	userID := r.URL.Query().Get("editUserID")

	id, err := strconv.Atoi(userID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	newUser, err := internal.GetUserByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	newUserUsername := r.URL.Query().Get("editUserUsername")
	if newUserUsername != "" {
		newUser.Username = newUserUsername
	}

	newUserPassword := r.URL.Query().Get("editUserPassword")
	if newUserPassword != "" {
		newUser.PasswordHash, err = internal.HashPassword(newUserPassword)
		if err != nil {
			return fmt.Errorf("internal error: %w", err)
		}
	}
	permissions.Transfers = r.URL.Query().Get("editUserPermissionsTransfers")
	permissions.Servers = r.URL.Query().Get("editUserPermissionsServers")
	permissions.Partners = r.URL.Query().Get("editUserPermissionsPartners")
	permissions.Rules = r.URL.Query().Get("editUserPermissionsRules")
	permissions.Users = r.URL.Query().Get("editUserPermissionsUsers")
	permissions.Administration = r.URL.Query().Get("editUserPermissionsAdministration")

	newUser.Permissions, err = model.PermsToMask(&permissions)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.UpdateUser(db, newUser); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func deleteUser(db *database.DB, r *http.Request) error {
	userID := r.URL.Query().Get("deleteUser")

	id, err := strconv.Atoi(userID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	newUser, err := internal.GetUserByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteUser(db, newUser); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
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

//nolint:funlen // pattern
func userManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("user_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		userList := listUser(db, r)

		var uPermissionsList []userPermissions
		for _, u := range userList {
			uPermissionsList = append(uPermissionsList, userPermissions{
				Username:    u.Username,
				ID:          u.ID,
				Permissions: model.MaskToPerms(u.Permissions),
			})
		}

		if r.Method == http.MethodGet && r.URL.Query().Get("newUserUsername") != "" {
			newUserErr := addUser(db, r)
			if newUserErr != nil {
				logger.Error("Internal error: %v", newUserErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		if r.Method == http.MethodGet && r.URL.Query().Get("editUserID") != "" {
			editUserErr := editUser(db, r)
			if editUserErr != nil {
				logger.Error("Internal error: %v", editUserErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		if r.Method == http.MethodGet && r.URL.Query().Get("deleteUser") != "" {
			deleteUserErr := deleteUser(db, r)
			if deleteUserErr != nil {
				logger.Error("Internal error: %v", deleteUserErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)

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
