//nolint:dupl // in process
package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
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
	urlParams := r.URL.Query()

	newUserUsername := urlParams.Get("newUserUsername")
	if newUserUsername != "" {
		newUser.Username = newUserUsername
	}

	newUserPassword := urlParams.Get("newUserPassword")
	if newUserPassword != "" {
		newUser.PasswordHash, err = internal.HashPassword(newUserPassword)
		if err != nil {
			return fmt.Errorf("failed to hash user password: %w", err)
		}
	}
	permissions.Transfers = strPermissions(urlParams["newUserPermissionsTransfers"])
	permissions.Servers = strPermissions(urlParams["newUserPermissionsServers"])
	permissions.Partners = strPermissions(urlParams["newUserPermissionsPartners"])
	permissions.Rules = strPermissions(urlParams["newUserPermissionsRules"])
	permissions.Users = strPermissions(urlParams["newUserPermissionsUsers"])
	permissions.Administration = strPermissions(urlParams["newUserPermissionsAdministration"])

	newUser.Permissions, err = model.PermsToMask(&permissions)
	if err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err = internal.InsertUser(db, &newUser); err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

func editUser(db *database.DB, r *http.Request) error {
	var permissions model.Permissions
	urlParams := r.URL.Query()
	userID := urlParams.Get("editUserID")

	id, err := strconv.Atoi(userID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	newUser, err := internal.GetUserByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	newUserUsername := urlParams.Get("editUserUsername")
	if newUserUsername != "" {
		newUser.Username = newUserUsername
	}

	newUserPassword := urlParams.Get("editUserPassword")
	if newUserPassword != "" {
		newUser.PasswordHash, err = internal.HashPassword(newUserPassword)
		if err != nil {
			return fmt.Errorf("failed to hash user password: %w", err)
		}
	}
	permissions.Transfers = strPermissions(urlParams["editUserPermissionsTransfers"])
	permissions.Servers = strPermissions(urlParams["editUserPermissionsServers"])
	permissions.Partners = strPermissions(urlParams["editUserPermissionsPartners"])
	permissions.Rules = strPermissions(urlParams["editUserPermissionsRules"])
	permissions.Users = strPermissions(urlParams["editUserPermissionsUsers"])
	permissions.Administration = strPermissions(urlParams["editUserPermissionsAdministration"])

	newUser.Permissions, err = model.PermsToMask(&permissions)
	if err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err = internal.UpdateUser(db, newUser); err != nil {
		return fmt.Errorf("failed to edit user: %w", err)
	}

	return nil
}

func deleteUser(db *database.DB, r *http.Request) error {
	userID := r.URL.Query().Get("deleteUser")

	id, err := strconv.Atoi(userID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	newUser, err := internal.GetUserByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	if err = internal.DeleteUser(db, newUser); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func listUser(db *database.DB, r *http.Request) ([]*model.User, Filters, string) {
	userFound := ""
	filter := Filters{
		Offset:          0,
		Limit:           LimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}
	urlParams := r.URL.Query()

	if urlParams.Get("orderAsc") == "true" {
		filter.OrderAsc = true
	} else if urlParams.Get("orderAsc") == "false" {
		filter.OrderAsc = false
	}

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := strconv.Atoi(limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := strconv.Atoi(offsetRes); err == nil {
			filter.Offset = o
		}
	}

	user, err := internal.ListUsers(db, "username", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, Filters{}, userFound
	}

	if search := urlParams.Get("search"); search != "" {
		if userSearch := searchUser(search, user); userSearch != nil {
			filter.DisableNext = true
			filter.DisablePrevious = true
			userFound = "true"

			return []*model.User{userSearch}, filter, userFound
		} else {
			userFound = "false"
		}
	}

	filter.Permissions = urlParams.Get("permissions")
	filter.PermissionsType = urlParams.Get("permissionsType")
	filter.PermissionsValue = urlParams.Get("permissionsValue")

	if filter.Permissions != "" && filter.PermissionsType != "" && filter.PermissionsValue != "" {
		user = permissionsFilter(filter.Permissions, filter.PermissionsType, filter.PermissionsValue, user)
	}

	users, filtersPtr := paginationFunc(r, user, &filter)

	return users, *filtersPtr, userFound
}

func paginationFunc(r *http.Request, user []*model.User, filter *Filters) ([]*model.User, *Filters) {
	nbUsers := len(user)
	urlParams := r.URL.Query()

	if urlParams.Get("previous") == "true" && filter.Offset > 0 {
		filter.Offset--
	}

	if urlParams.Get("next") == "true" {
		if filter.Limit*(filter.Offset+1) <= nbUsers {
			filter.Offset++
		}
	}

	start := filter.Offset * filter.Limit
	end := start + filter.Limit

	if start > nbUsers {
		start = nbUsers
	}

	if end > nbUsers {
		end = nbUsers
	}

	pagedUsers := user[start:end]

	if filter.Offset == 0 {
		filter.DisablePrevious = true
	}

	if (filter.Offset+1)*filter.Limit >= nbUsers {
		filter.DisableNext = true
	}

	return pagedUsers, filter
}

func permissionsFilter(filterUser, filterUserType, filterUserValue string, listU []*model.User) []*model.User {
	switch filterUserType {
	case "permissionsEq":
		return permissionsFilterLoop(filterUser, filterUserValue, listU)
	case "permissionsMin":
		var userFiltered []*model.User
		for _, val := range filterMin(filterUserValue) {
			userFiltered = append(userFiltered, permissionsFilterLoop(filterUser, val, listU)...)
		}

		return userFiltered
	case "permissionsMax":
		var userFiltered []*model.User
		for _, val := range filterMax(filterUserValue) {
			userFiltered = append(userFiltered, permissionsFilterLoop(filterUser, val, listU)...)
		}

		return userFiltered
	default:
		return nil
	}
}

func filterMin(filterUserValue string) []string {
	switch filterUserValue {
	case "---":
		return []string{"---", "r--", "rw-", "rwd"}
	case "r--":
		return []string{"r--", "rw-", "rwd"}
	case "rw-":
		return []string{"rw-", "rwd"}
	case "rwd":
		return []string{"rwd"}
	default:
		return nil
	}
}

func filterMax(filterUserValue string) []string {
	switch filterUserValue {
	case "rwd":
		return []string{"---", "r--", "rw-", "rwd"}
	case "rw-":
		return []string{"---", "r--", "rw-"}
	case "r--":
		return []string{"---", "r--"}
	case "---":
		return []string{"---"}
	default:
		return nil
	}
}

//nolint:gocognit // permissionsFilterLoop
func permissionsFilterLoop(filterUser, filterUserValue string, listU []*model.User) []*model.User {
	userFiltered := make([]*model.User, 0, len(listU))

	for _, uModel := range listU {
		perms := model.MaskToPerms(uModel.Permissions)

		switch filterUser {
		case "filterPermissionsTransfers":
			if perms.Transfers == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		case "filterPermissionsServers":
			if perms.Servers == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		case "filterPermissionsPartners":
			if perms.Partners == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		case "filterPermissionsRules":
			if perms.Rules == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		case "filterPermissionsUsers":
			if perms.Users == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		case "filterPermissionsAdministration":
			if perms.Administration == filterUserValue {
				userFiltered = append(userFiltered, uModel)
			}
		}
	}

	return userFiltered
}

func autocompletionFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")

		users, err := internal.GetUsersLike(db, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(users))
		for i, u := range users {
			names[i] = u.Username
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(names); err != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchUser(userNameSearch string, listUserSearch []*model.User) *model.User {
	for _, u := range listUserSearch {
		if u.Username == userNameSearch {
			return u
		}
	}

	return nil
}

func callMethodsUserManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request) {
	urlParams := r.URL.Query()
	if r.Method == http.MethodGet && urlParams.Get("newUserUsername") != "" {
		if newUserErr := addUser(db, r); newUserErr != nil {
			logger.Error("failed to add user: %v", newUserErr)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return
	}

	if r.Method == http.MethodGet && urlParams.Get("editUserID") != "" {
		if editUserErr := editUser(db, r); editUserErr != nil {
			logger.Error("failed to edit user: %v", editUserErr)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return
	}

	if r.Method == http.MethodGet && urlParams.Get("deleteUser") != "" {
		if deleteUserErr := deleteUser(db, r); deleteUserErr != nil {
			logger.Error("failed to delete user: %v", deleteUserErr)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return
	}
}

//nolint:funlen // pattern
func userManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("user_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		userList, filter, userFound := listUser(db, r)

		var uPermissionsList []userPermissions
		for _, u := range userList {
			uPermissionsList = append(uPermissionsList, userPermissions{
				Username:    u.Username,
				ID:          u.ID,
				Permissions: model.MaskToPerms(u.Permissions),
			})
		}

		callMethodsUserManagement(logger, db, w, r)

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("failed to get user by token: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if err := userManagementTemplate.ExecuteTemplate(w, "user_management_page", map[string]any{
			"userPermissions": uPermissionsList,
			"myPermission":    myPermission,
			"tab":             tabTranslated,
			"username":        user.Username,
			"language":        userLanguage,
			"userFound":       userFound,
			"filter":          filter,
			"currentPage":     currentPage,
		}); err != nil {
			logger.Error("render user_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
