//nolint:dupl // in process
package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
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

	if errForm := r.ParseForm(); errForm != nil {
		return fmt.Errorf("failed to parse form: %w", errForm)
	}

	newUserUsername := r.FormValue("newUserUsername")
	if newUserUsername != "" {
		newUser.Username = newUserUsername
	}

	newUserPassword := r.FormValue("newUserPassword")
	if newUserPassword != "" {
		newUser.PasswordHash, err = internal.HashPassword(newUserPassword)
		if err != nil {
			return fmt.Errorf("failed to hash user password: %w", err)
		}
	}
	permissions.Transfers = strPermissions(r.Form["newUserPermissionsTransfers[]"])
	permissions.Servers = strPermissions(r.Form["newUserPermissionsServers[]"])
	permissions.Partners = strPermissions(r.Form["newUserPermissionsPartners[]"])
	permissions.Rules = strPermissions(r.Form["newUserPermissionsRules[]"])
	permissions.Users = strPermissions(r.Form["newUserPermissionsUsers[]"])
	permissions.Administration = strPermissions(r.Form["newUserPermissionsAdministration[]"])

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

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	userID := r.FormValue("editUserID")

	id, err := internal.ParseUint[uint64](userID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	newUser, err := internal.GetUserByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	newUserUsername := r.FormValue("editUserUsername")
	if newUserUsername != "" {
		newUser.Username = newUserUsername
	}

	newUserPassword := r.FormValue("editUserPassword")
	if newUserPassword != "" {
		newUser.PasswordHash, err = internal.HashPassword(newUserPassword)
		if err != nil {
			return fmt.Errorf("failed to hash user password: %w", err)
		}
	}
	permissions.Transfers = strPermissions(r.Form["editUserPermissionsTransfers[]"])
	permissions.Servers = strPermissions(r.Form["editUserPermissionsServers[]"])
	permissions.Partners = strPermissions(r.Form["editUserPermissionsPartners[]"])
	permissions.Rules = strPermissions(r.Form["editUserPermissionsRules[]"])
	permissions.Users = strPermissions(r.Form["editUserPermissionsUsers[]"])
	permissions.Administration = strPermissions(r.Form["editUserPermissionsAdministration[]"])

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
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	userID := r.FormValue("deleteUser")

	id, err := internal.ParseUint[uint64](userID)
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

func listUser(db *database.DB, r *http.Request) ([]*model.User, *Filters, string) {
	userFound := ""
	defaultFilter := &Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "user_management_page"); ok {
		filter = saved
	}

	isApply := r.URL.Query().Get("applyFilters") == True
	if isApply {
		filter = defaultFilter
	}

	urlParams := r.URL.Query()

	if urlParams.Get("orderAsc") != "" {
		filter.OrderAsc = urlParams.Get("orderAsc") == True
	}

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := internal.ParseUint[uint64](limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := internal.ParseUint[uint64](offsetRes); err == nil {
			filter.Offset = o
		}
	}

	user, err := internal.ListUsers(db, "username", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, nil, userFound
	}

	if search := urlParams.Get("search"); search != "" {
		if userSearch := searchUser(search, user); userSearch != nil {
			filter.DisableNext = true
			filter.DisablePrevious = true
			userFound = True

			return []*model.User{userSearch}, filter, userFound
		}
		userFound = False
	}

	hasPermParams := urlParams.Has("permissions") || urlParams.Has("permissionsType") || urlParams.Has("permissionsValue")
	if isApply || hasPermParams {
		filter.Permissions = urlParams.Get("permissions")
		filter.PermissionsType = urlParams.Get("permissionsType")
		filter.PermissionsValue = urlParams.Get("permissionsValue")
	}

	if filter.Permissions != "" && filter.PermissionsType != "" && filter.PermissionsValue != "" {
		user = permissionsFilter(filter.Permissions, filter.PermissionsType, filter.PermissionsValue, user)
	}

	users, filtersPtr := paginationFunc(r, user, filter)

	return users, filtersPtr, userFound
}

func paginationFunc(r *http.Request, user []*model.User, filter *Filters) ([]*model.User, *Filters) {
	nbUsers := uint64(len(user))
	urlParams := r.URL.Query()

	if urlParams.Get("previous") == True && filter.Offset > 0 {
		filter.Offset--
	}

	if urlParams.Get("next") == True {
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

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
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

func callMethodsUserManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("newUserUsername") != "" {
		if newUserErr := addUser(db, r); newUserErr != nil {
			logger.Errorf("failed to add user: %v", newUserErr)
			modalElement = getFormValues(r)

			return false, newUserErr.Error(), "addUserModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editUserID") != "" {
		idEdit := r.FormValue("editUserID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		if editUserErr := editUser(db, r); editUserErr != nil {
			logger.Errorf("failed to edit user: %v", editUserErr)
			modalElement = getFormValues(r)

			return false, editUserErr.Error(), fmt.Sprintf("editUserModal_%d", id), modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("deleteUser") != "" {
		if deleteUserErr := deleteUser(db, r); deleteUserErr != nil {
			logger.Errorf("failed to delete user: %v", deleteUserErr)

			return false, deleteUserErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

//nolint:funlen // pattern
func userManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		userLanguage := locale.GetLanguage(r)
		tabTranslated := pageTranslated("user_management_page", userLanguage)
		userList, filter, userFound := listUser(db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "user_management_page", filter)

		var uPermissionsList []userPermissions
		for _, u := range userList {
			uPermissionsList = append(uPermissionsList, userPermissions{
				Username:    u.Username,
				ID:          u.ID,
				Permissions: model.MaskToPerms(u.Permissions),
			})
		}

		value, errMsg, modalOpen, modalElement := callMethodsUserManagement(logger, db, w, r)
		if value {
			return
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := userManagementTemplate.ExecuteTemplate(w, "user_management_page", map[string]any{
			"appName":         constants.AppName,
			"version":         version.Num,
			"compileDate":     version.Date,
			"revision":        version.Commit,
			"docLink":         constants.DocLink(userLanguage),
			"userPermissions": uPermissionsList,
			"myPermission":    myPermission,
			"tab":             tabTranslated,
			"username":        user.Username,
			"language":        userLanguage,
			"userFound":       userFound,
			"filter":          filter,
			"currentPage":     currentPage,
			"errMsg":          errMsg,
			"modalOpen":       modalOpen,
			"modalElement":    modalElement,
		}); tmplErr != nil {
			logger.Errorf("render user_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
