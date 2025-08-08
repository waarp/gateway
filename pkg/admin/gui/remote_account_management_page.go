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
func listRemoteAccount(partnerName string, db *database.DB, r *http.Request) (
	[]*model.RemoteAccount, FiltersPagination, string,
) {
	remoteAccountFound := ""
	filter := FiltersPagination{
		Offset:          0,
		Limit:           DefaultLimitPagination,
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
		if l, err := strconv.ParseUint(limitRes, 10, 64); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := strconv.ParseUint(offsetRes, 10, 64); err == nil {
			filter.Offset = o
		}
	}

	remotesAccounts, err := internal.ListPartnerAccounts(db, partnerName, "login", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, remoteAccountFound
	}

	if search := urlParams.Get("search"); search != "" && searchRemoteAccount(search, remotesAccounts) == nil {
		remoteAccountFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		remoteAccountFound = "true"

		return []*model.RemoteAccount{searchRemoteAccount(search, remotesAccounts)}, filter, remoteAccountFound
	}

	paginationPage(&filter, uint64(len(remoteAccountFound)), r)

	remotesAccountsList, err := internal.ListPartnerAccounts(db, partnerName, "login",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, FiltersPagination{}, remoteAccountFound
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
			id, err = strconv.ParseUint(partnerID, 10, 64)
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

	id, err := strconv.ParseUint(remoteAccountID, 10, 64)
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

	id, err := strconv.ParseUint(remoteAccountID, 10, 64)
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
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("deleteRemoteAccount") != "" {
		deleteRemoteAccountErr := deleteRemoteAccount(db, r)
		if deleteRemoteAccountErr != nil {
			logger.Errorf("failed to delete remote account: %v", deleteRemoteAccountErr)

			return false, deleteRemoteAccountErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("addRemoteAccountLogin") != "" {
		addRemoteAccountErr := addRemoteAccount(partner.Name, db, r)
		if addRemoteAccountErr != nil {
			logger.Errorf("failed to add remote account: %v", addRemoteAccountErr)

			return false, addRemoteAccountErr.Error(), "addRemoteAccountModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editRemoteAccountID") != "" {
		idEdit := r.FormValue("editRemoteAccountID")

		id, err := strconv.ParseUint(idEdit, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", ""
		}

		editRemoteAccountErr := editRemoteAccount(db, r)
		if editRemoteAccountErr != nil {
			logger.Errorf("failed to edit remote account: %v", editRemoteAccountErr)

			return false, editRemoteAccountErr.Error(), fmt.Sprintf("editRemoteAccountModal_%d", id)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func remoteAccountPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("remote_account_management_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var partner *model.RemoteAgent
		var id uint64

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			id, err = strconv.ParseUint(partnerID, 10, 64)
			if err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		remoteAccounts, filter, remoteAccountFound := listRemoteAccount(partner.Name, db, r)

		value, errMsg, modalOpen := callMethodsRemoteAccount(logger, db, w, r, partner)
		if value {
			return
		}

		currentPage := filter.Offset + 1

		if tmplErr := remoteAccountTemplate.ExecuteTemplate(w, "remote_account_management_page", map[string]any{
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
			"hasPartnerID":       true,
		}); tmplErr != nil {
			logger.Errorf("render partner_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
