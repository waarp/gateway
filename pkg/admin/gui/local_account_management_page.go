//nolint:dupl // method local account management
package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl // it is not the same function, the calls are different
func listLocalAccount(serverName string, db *database.DB, r *http.Request) (
	[]*model.LocalAccount, Filters, string,
) {
	localAccountFound := ""
	defaultFilter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "local_account_management_page"); ok {
		filter = saved
	}

	if r.URL.Query().Get("applyFilters") == True {
		filter = defaultFilter
	}

	urlParams := r.URL.Query()

	if urlParams.Get("orderAsc") != "" {
		filter.OrderAsc = urlParams.Get("orderAsc") == True
	}

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := strconv.ParseUint(limitRes, 10, 64); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := strconv.ParseUint(offsetRes, 10, 64); err == nil {
			filter.Offset = o
		}
	}

	localsAccounts, err := internal.ListServerAccounts(db, serverName, "login", true, 0, 0)
	if err != nil {
		return nil, Filters{}, localAccountFound
	}

	if search := urlParams.Get("search"); search != "" && searchLocalAccount(search, localsAccounts) == nil {
		localAccountFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		localAccountFound = True

		return []*model.LocalAccount{searchLocalAccount(search, localsAccounts)}, filter, localAccountFound
	}

	paginationPage(&filter, uint64(len(localAccountFound)), r)

	localsAccountsList, err := internal.ListServerAccounts(db, serverName, "login",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, localAccountFound
	}

	return localsAccountsList, filter, localAccountFound
}

func autocompletionLocalAccountFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		prefix := urlParams.Get("q")
		var err error
		var server *model.LocalAgent
		var id uint64

		serverID := urlParams.Get("serverID")
		if serverID != "" {
			id, err = strconv.ParseUint(serverID, 10, 64)
			if err != nil {
				http.Error(w, "failed to convert id to int", http.StatusInternalServerError)

				return
			}

			server, err = internal.GetServerByID(db, int64(id))
			if err != nil {
				http.Error(w, "failed to get id", http.StatusInternalServerError)

				return
			}
		}

		localAccounts, err := internal.GetServerAccountsLike(db, server.Name, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(localAccounts))
		for i, u := range localAccounts {
			names[i] = u.Login
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchLocalAccount(localAccountLoginSearch string,
	listLocalAccountSearch []*model.LocalAccount,
) *model.LocalAccount {
	for _, la := range listLocalAccountSearch {
		if la.Login == localAccountLoginSearch {
			return la
		}
	}

	return nil
}

func editLocalAccount(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	localAccountID := r.FormValue("editLocalAccountID")

	id, err := strconv.ParseUint(localAccountID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editLocalAccount, err := internal.GetServerAccountByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get local account: %w", err)
	}

	if editLocalAccountLogin := r.FormValue("editLocalAccountLogin"); editLocalAccountLogin != "" {
		editLocalAccount.Login = editLocalAccountLogin
	}

	if err = internal.UpdateServerAccount(db, editLocalAccount); err != nil {
		return fmt.Errorf("failed to edit local account: %w", err)
	}

	return nil
}

func addLocalAccount(serverName string, db *database.DB, r *http.Request) error {
	var newLocalAccount model.LocalAccount

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newLocalAccountLogin := r.FormValue("addLocalAccountLogin"); newLocalAccountLogin != "" {
		newLocalAccount.Login = newLocalAccountLogin
	}

	server, err := internal.GetServer(db, serverName)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	newLocalAccount.LocalAgentID = server.ID

	if addErr := internal.InsertServerAccount(db, &newLocalAccount); addErr != nil {
		return fmt.Errorf("failed to add local account: %w", addErr)
	}

	return nil
}

//nolint:dupl // it is not the same function, the calls are different
func deleteLocalAccount(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	localAccountID := r.FormValue("deleteLocalAccount")

	id, err := strconv.ParseUint(localAccountID, 10, 64)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	localAccount, err := internal.GetServerAccountByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteServerAccount(db, localAccount); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func callMethodsLocalAccount(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	server *model.LocalAgent,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("deleteLocalAccount") != "" {
		deleteLocalAccountErr := deleteLocalAccount(db, r)
		if deleteLocalAccountErr != nil {
			logger.Errorf("failed to delete local account: %v", deleteLocalAccountErr)

			return false, deleteLocalAccountErr.Error(), "", nil
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d", r.URL.Path, server.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("addLocalAccountLogin") != "" {
		addLocalAccountErr := addLocalAccount(server.Name, db, r)
		if addLocalAccountErr != nil {
			logger.Errorf("failed to add local account: %v", addLocalAccountErr)
			modalElement = getFormValues(r)

			return false, addLocalAccountErr.Error(), "addLocalAccountModal", modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d", r.URL.Path, server.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editLocalAccountID") != "" {
		idEdit := r.FormValue("editLocalAccountID")

		id, err := strconv.ParseUint(idEdit, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		editLocalAccountErr := editLocalAccount(db, r)
		if editLocalAccountErr != nil {
			logger.Errorf("failed to edit local account: %v", editLocalAccountErr)
			modalElement = getFormValues(r)

			return false, editLocalAccountErr.Error(), fmt.Sprintf("editLocalAccountModal_%d", id), modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d", r.URL.Path, server.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func localAccountPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("local_account_management_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var server *model.LocalAgent
		var id uint64

		serverID := r.URL.Query().Get("serverID")
		if serverID != "" {
			id, err = strconv.ParseUint(serverID, 10, 64)
			if err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			server, err = internal.GetServerByID(db, int64(id))
			if err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		localAccounts, filter, localAccountFound := listLocalAccount(server.Name, db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "local_account_management_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsLocalAccount(logger, db, w, r, server)
		if value {
			return
		}

		currentPage := filter.Offset + 1

		if tmplErr := localAccountTemplate.ExecuteTemplate(w, "local_account_management_page", map[string]any{
			"myPermission":      myPermission,
			"username":          user.Username,
			"language":          userLanguage,
			"server":            server,
			"localAccounts":     localAccounts,
			"filter":            filter,
			"currentPage":       currentPage,
			"localAccountFound": localAccountFound,
			"tab":               tTranslated,
			"errMsg":            errMsg,
			"modalOpen":         modalOpen,
			"modalElement":      modalElement,
			"hasServerID":       true,
			"sidebarSection":    "connection",
			"sidebarLink":       "server_management",
		}); tmplErr != nil {
			logger.Errorf("render server_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
