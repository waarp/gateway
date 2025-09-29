//nolint:dupl // method remote account management
package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:dupl // it is not the same function, the calls are different
func listRemoteAccount(partnerName string, db *database.DB, r *http.Request) (
	[]*model.RemoteAccount, Filters, string,
) {
	remoteAccountFound := ""
	defaultFilter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "remote_account_management_page"); ok {
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
		if l, err := internal.ParseUint[uint64](limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := internal.ParseUint[uint64](offsetRes); err == nil {
			filter.Offset = o
		}
	}

	remotesAccounts, err := internal.ListPartnerAccounts(db, partnerName, "login", true, 0, 0)
	if err != nil {
		return nil, Filters{}, remoteAccountFound
	}

	if search := urlParams.Get("search"); search != "" && searchRemoteAccount(search, remotesAccounts) == nil {
		remoteAccountFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		remoteAccountFound = True

		return []*model.RemoteAccount{searchRemoteAccount(search, remotesAccounts)}, filter, remoteAccountFound
	}

	paginationPage(&filter, uint64(len(remoteAccountFound)), r)

	remotesAccountsList, err := internal.ListPartnerAccounts(db, partnerName, "login",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, remoteAccountFound
	}

	return remotesAccountsList, filter, remoteAccountFound
}

func autocompletionRemoteAccountFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		prefix := urlParams.Get("q")
		var err error
		var partner *model.RemoteAgent
		var id uint64

		partnerID := urlParams.Get("partnerID")
		if partnerID != "" {
			id, err = internal.ParseUint[uint64](partnerID)
			if err != nil {
				http.Error(w, "failed to convert id to int", http.StatusInternalServerError)

				return
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				http.Error(w, "failed to get id", http.StatusInternalServerError)

				return
			}
		}

		remoteAccounts, err := internal.GetPartnerAccountsLike(db, partner.Name, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(remoteAccounts))
		for i, u := range remoteAccounts {
			names[i] = u.Login
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchRemoteAccount(remoteAccountLoginSearch string,
	listRemoteAccountSearch []*model.RemoteAccount,
) *model.RemoteAccount {
	for _, ra := range listRemoteAccountSearch {
		if ra.Login == remoteAccountLoginSearch {
			return ra
		}
	}

	return nil
}

func editRemoteAccount(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	remoteAccountID := r.FormValue("editRemoteAccountID")

	id, err := internal.ParseUint[uint64](remoteAccountID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editRemoteAccount, err := internal.GetPartnerAccountByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get remote account: %w", err)
	}

	if editRemoteAccountLogin := r.FormValue("editRemoteAccountLogin"); editRemoteAccountLogin != "" {
		editRemoteAccount.Login = editRemoteAccountLogin
	}

	if err = internal.UpdatePartnerAccount(db, editRemoteAccount); err != nil {
		return fmt.Errorf("failed to edit remote account: %w", err)
	}

	return nil
}

func addRemoteAccount(partnerName string, db *database.DB, r *http.Request) error {
	var newRemoteAccount model.RemoteAccount

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newRemoteAccountLogin := r.FormValue("addRemoteAccountLogin"); newRemoteAccountLogin != "" {
		newRemoteAccount.Login = newRemoteAccountLogin
	}

	partner, err := internal.GetPartner(db, partnerName)
	if err != nil {
		return fmt.Errorf("failed to get partner: %w", err)
	}

	newRemoteAccount.RemoteAgentID = partner.ID

	if addErr := internal.InsertPartnerAccount(db, &newRemoteAccount); addErr != nil {
		return fmt.Errorf("failed to add remote account: %w", addErr)
	}

	return nil
}

//nolint:dupl // it is not the same function, the calls are different
func deleteRemoteAccount(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	remoteAccountID := r.FormValue("deleteRemoteAccount")

	id, err := internal.ParseUint[uint64](remoteAccountID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	remoteAccount, err := internal.GetPartnerAccountByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeletePartnerAccount(db, remoteAccount); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func callMethodsRemoteAccount(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	partner *model.RemoteAgent,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("deleteRemoteAccount") != "" {
		deleteRemoteAccountErr := deleteRemoteAccount(db, r)
		if deleteRemoteAccountErr != nil {
			logger.Errorf("failed to delete remote account: %v", deleteRemoteAccountErr)

			return false, deleteRemoteAccountErr.Error(), "", nil
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("addRemoteAccountLogin") != "" {
		addRemoteAccountErr := addRemoteAccount(partner.Name, db, r)
		if addRemoteAccountErr != nil {
			logger.Errorf("failed to add remote account: %v", addRemoteAccountErr)
			modalElement = getFormValues(r)

			return false, addRemoteAccountErr.Error(), "addRemoteAccountModal", modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editRemoteAccountID") != "" {
		idEdit := r.FormValue("editRemoteAccountID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		editRemoteAccountErr := editRemoteAccount(db, r)
		if editRemoteAccountErr != nil {
			logger.Errorf("failed to edit remote account: %v", editRemoteAccountErr)
			modalElement = getFormValues(r)

			return false, editRemoteAccountErr.Error(), fmt.Sprintf("editRemoteAccountModal_%d", id), modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func remoteAccountPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := locale.GetLanguage(r)
		tTranslated := pageTranslated("remote_account_management_page", userLanguage)

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var partner *model.RemoteAgent
		var id uint64

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			id, err = internal.ParseUint[uint64](partnerID)
			if err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		remoteAccounts, filter, remoteAccountFound := listRemoteAccount(partner.Name, db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "remote_account_management_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsRemoteAccount(logger, db, w, r, partner)
		if value {
			return
		}

		currentPage := filter.Offset + 1

		if tmplErr := remoteAccountTemplate.ExecuteTemplate(w, "remote_account_management_page", map[string]any{
			"appName":            constants.AppName,
			"version":            version.Num,
			"compileDate":        version.Date,
			"revision":           version.Commit,
			"docLink":            constants.DocLink(userLanguage),
			"myPermission":       myPermission,
			"username":           user.Username,
			"language":           userLanguage,
			"partner":            partner,
			"remoteAccounts":     remoteAccounts,
			"filter":             filter,
			"currentPage":        currentPage,
			"remoteAccountFound": remoteAccountFound,
			"tab":                tTranslated,
			"errMsg":             errMsg,
			"modalOpen":          modalOpen,
			"modalElement":       modalElement,
			"hasPartnerID":       true,
			"sidebarSection":     "connection",
			"sidebarLink":        "partner_management",
		}); tmplErr != nil {
			logger.Errorf("render partner_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
