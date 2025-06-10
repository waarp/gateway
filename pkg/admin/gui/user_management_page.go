//nolint:dupl // in process
package gui

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var userFound = true //nolint:gochecknoglobals //userFound

type userPermissions struct {
	Username    string
	ID          int64
	Permissions *model.Permissions
}

func strPermissions(p []string) string {
	res := ""
	if slices.Contains(p, "r") {
		res += "r"
	} else {
		res += "-"
	}

	if slices.Contains(p, "w") {
		res += "w"
	} else {
		res += "-"
	}

	if slices.Contains(p, "d") {
		res += "d"
	} else {
		res += "-"
	}

	return res
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
	permissions.Transfers = strPermissions(r.URL.Query()["newUserPermissionsTransfers"])
	permissions.Servers = strPermissions(r.URL.Query()["newUserPermissionsServers"])
	permissions.Partners = strPermissions(r.URL.Query()["newUserPermissionsPartners"])
	permissions.Rules = strPermissions(r.URL.Query()["newUserPermissionsRules"])
	permissions.Users = strPermissions(r.URL.Query()["newUserPermissionsUsers"])
	permissions.Administration = strPermissions(r.URL.Query()["newUserPermissionsAdministration"])

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
	permissions.Transfers = strPermissions(r.URL.Query()["editUserPermissionsTransfers"])
	permissions.Servers = strPermissions(r.URL.Query()["editUserPermissionsServers"])
	permissions.Partners = strPermissions(r.URL.Query()["editUserPermissionsPartners"])
	permissions.Rules = strPermissions(r.URL.Query()["editUserPermissionsRules"])
	permissions.Users = strPermissions(r.URL.Query()["editUserPermissionsUsers"])
	permissions.Administration = strPermissions(r.URL.Query()["editUserPermissionsAdministration"])

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
	userFound = true
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

	search := r.URL.Query().Get("search")
	if search != "" {
		userSearch := searchUser(search, user)
		if userSearch != nil {
			return []*model.User{userSearch}
		} else {
			userFound = false
		}
	}

	filterUser := r.URL.Query().Get("permissions")
	filterUserType := r.URL.Query().Get("permissionsType")
	filterUserValue := r.URL.Query().Get("permissionsValue")

	if filterUser != "" && filterUserType != "" && filterUserValue != "" {
		return permissionsFilter(filterUser, filterUserType, filterUserValue, user)
	}

	return user
}

func permissionsFilter(filterUser, filterUserType, filterUserValue string, listU []*model.User) []*model.User {
	if filterUserType == "permissionsEq" {
		return permissionsFilterLoop(filterUser, filterUserValue, listU)
	}

	if filterUserType == "permissionsMin" {
		var userFiltered []*model.User
		for _, val := range filterMin(filterUserValue) {
			userFiltered = append(userFiltered, permissionsFilterLoop(filterUser, val, listU)...)
		}

		return userFiltered
	}

	if filterUserType == "permissionsMax" {
		var userFiltered []*model.User
		for _, val := range filterMax(filterUserValue) {
			userFiltered = append(userFiltered, permissionsFilterLoop(filterUser, val, listU)...)
		}

		return userFiltered
	}

	return nil
}

func filterMin(filterUserValue string) []string {
	if filterUserValue == "---" {
		return []string{"---", "r--", "rw-", "rwd"}
	}

	if filterUserValue == "r--" {
		return []string{"r--", "rw-", "rwd"}
	}

	if filterUserValue == "rw-" {
		return []string{"rw-", "rwd"}
	}

	if filterUserValue == "rwd" {
		return []string{"rwd"}
	}

	return nil
}

func filterMax(filterUserValue string) []string {
	if filterUserValue == "rwd" {
		return []string{"---", "r--", "rw-", "rwd"}
	}

	if filterUserValue == "rw-" {
		return []string{"---", "r--", "rw-"}
	}

	if filterUserValue == "r--" {
		return []string{"---", "r--"}
	}

	if filterUserValue == "---" {
		return []string{"---"}
	}

	return nil
}

//nolint:gocognit // permissionsFilterLoop
func permissionsFilterLoop(filterUser, filterUserValue string, listU []*model.User) []*model.User {
	var userFiltered []*model.User

	for _, uModel := range listU {
		perms := model.MaskToPerms(uModel.Permissions)
		if filterUser == "filterPermissionsTransfers" {
			if perms.Transfers == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}

		if filterUser == "filterPermissionsServers" {
			if perms.Servers == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}

		if filterUser == "filterPermissionsPartners" {
			if perms.Partners == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}

		if filterUser == "filterPermissionsRules" {
			if perms.Rules == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}

		if filterUser == "filterPermissionsUsers" {
			if perms.Users == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}

		if filterUser == "filterPermissionsAdministration" {
			if perms.Administration == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}
	}

	return userFiltered
}

func searchUser(userNameSearch string, listUserSearch []*model.User) *model.User {
	for _, u := range listUserSearch {
		if u.Username == userNameSearch {
			return u
		}
	}

	return nil
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
			"userFound":       userFound,
		}); err != nil {
			logger.Error("render user_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
