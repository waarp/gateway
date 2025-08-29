package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

//nolint:dupl // no similar func (is for account)
func listCredentialLocalAccount(serverName, login string, db *database.DB, r *http.Request) (
	[]*model.Credential, Filters, string,
) {
	credentialAccountFound := ""
	defaultFilter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "local_account_authentication_page"); ok {
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

	accountsCredentials, err := internal.ListServerAccountCredentials(db, serverName, login, "name", true, 0, 0)
	if err != nil {
		return nil, Filters{}, credentialAccountFound
	}

	if search := urlParams.Get("search"); search != "" && searchCredentialLocalAccount(search,
		accountsCredentials) == nil {
		credentialAccountFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialAccountFound = True

		return []*model.Credential{searchCredentialLocalAccount(search, accountsCredentials)}, filter, credentialAccountFound
	}

	paginationPage(&filter, uint64(len(accountsCredentials)), r)

	accountsCredentialsList, err := internal.ListServerAccountCredentials(db, serverName, login, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, credentialAccountFound
	}

	return accountsCredentialsList, filter, credentialAccountFound
}

//nolint:dupl // is not the same, GetCredentialsLike is called
func autocompletionCredentialsLocalAccountsFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		prefix := urlParams.Get("q")
		var err error
		var account *model.LocalAccount
		var idA uint64

		accountID := urlParams.Get("accountID")

		if accountID != "" {
			idA, err = internal.ParseUint[uint64](accountID)
			if err != nil {
				http.Error(w, "failed to convert account id to int", http.StatusInternalServerError)

				return
			}

			account, err = internal.GetServerAccountByID(db, int64(idA))
			if err != nil {
				http.Error(w, "failed to get account by id", http.StatusInternalServerError)

				return
			}
		}

		credentialsAccounts, err := internal.GetCredentialsLike(db, account, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(credentialsAccounts))
		for i, u := range credentialsAccounts {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchCredentialLocalAccount(credentialAccountNameSearch string,
	listCredentialAccountSearch []*model.Credential,
) *model.Credential {
	for _, cla := range listCredentialAccountSearch {
		if cla.Name == credentialAccountNameSearch {
			return cla
		}
	}

	return nil
}

func addCredentialLocalAccount(serverName, login string, db *database.DB, r *http.Request) error {
	var newCredentialAccount model.Credential

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newCredentialAccountName := r.FormValue("addCredentialAccountName"); newCredentialAccountName != "" {
		newCredentialAccount.Name = newCredentialAccountName
	}

	if newCredentialAccountType := r.FormValue("addCredentialAccountType"); newCredentialAccountType != "" {
		newCredentialAccount.Type = newCredentialAccountType
	}

	switch newCredentialAccount.Type {
	case auth.Password:
		newCredentialAccount.Value = r.FormValue("addCredentialValue")
	case auth.TLSTrustedCertificate, sftp.AuthSSHPublicKey:
		newCredentialAccount.Value = r.FormValue("addCredentialValueFile")
	}

	account, err := internal.GetServerAccount(db, serverName, login)
	if err != nil {
		return fmt.Errorf("failed to get server account: %w", err)
	}

	account.SetCredOwner(&newCredentialAccount)

	if addErr := internal.InsertCredential(db, &newCredentialAccount); addErr != nil {
		return fmt.Errorf("failed to add credential account: %w", addErr)
	}

	return nil
}

//nolint:dupl // no similar func (is for account)
func editCredentialLocalAccount(account *model.LocalAccount, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialAccountID := r.FormValue("editCredentialAccountID")

	id, err := internal.ParseUint[uint64](credentialAccountID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editCredentialAccount, err := internal.GetCredentialByID(db, account, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get credential account: %w", err)
	}

	if editCredentialAccountName := r.FormValue("editCredentialAccountName"); editCredentialAccountName != "" {
		editCredentialAccount.Name = editCredentialAccountName
	}

	if editCredentialAccountType := r.FormValue("editCredentialAccountType"); editCredentialAccountType != "" {
		editCredentialAccount.Type = editCredentialAccountType
	}

	switch editCredentialAccount.Type {
	case auth.Password:
		editCredentialAccount.Value = r.FormValue("editCredentialValue")
	case auth.TLSTrustedCertificate, sftp.AuthSSHPublicKey:
		editCredentialAccount.Value = r.FormValue("editCredentialValueFile")
	}

	if err = internal.UpdateCredential(db, editCredentialAccount); err != nil {
		return fmt.Errorf("failed to edit credential account: %w", err)
	}

	return nil
}

func deleteCredentialLocalAccount(account *model.LocalAccount, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialAccountID := r.FormValue("deleteCredentialAccount")

	id, err := internal.ParseUint[uint64](credentialAccountID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	credentialAccount, err := internal.GetCredentialByID(db, account, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteCredential(db, credentialAccount); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

//nolint:dupl // method local account authentication
func callMethodsLocalAccountAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	server *model.LocalAgent, account *model.LocalAccount,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("deleteCredentialAccount") != "" {
		deleteCredentialAccountErr := deleteCredentialLocalAccount(account, db, r)
		if deleteCredentialAccountErr != nil {
			logger.Errorf("failed to delete credential account: %v", deleteCredentialAccountErr)

			return false, deleteCredentialAccountErr.Error(), "", nil
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d&accountID=%d", r.URL.Path, server.ID, account.ID),
			http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("addCredentialAccountName") != "" {
		addCredentialAccountErr := addCredentialLocalAccount(server.Name, account.Login, db, r)
		if addCredentialAccountErr != nil {
			logger.Errorf("failed to add credential account: %v", addCredentialAccountErr)
			modalElement = getFormValues(r)

			return false, addCredentialAccountErr.Error(), "addCredentialAccountModal", modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d&accountID=%d", r.URL.Path, server.ID, account.ID),
			http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editCredentialAccountID") != "" {
		idEdit := r.FormValue("editCredentialAccountID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		editCredentialAccountErr := editCredentialLocalAccount(account, db, r)
		if editCredentialAccountErr != nil {
			logger.Errorf("failed to edit credential account: %v", editCredentialAccountErr)
			modalElement = getFormValues(r)

			return false, editCredentialAccountErr.Error(), fmt.Sprintf("editCredentialInternalModal_%d", id), modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d&accountID=%d", r.URL.Path, server.ID, account.ID),
			http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func getServerAndAccount(db *database.DB, serverID, accountID string, logger *log.Logger) (
	*model.LocalAgent, *model.LocalAccount,
) {
	var server *model.LocalAgent
	var account *model.LocalAccount

	if serverID != "" && accountID != "" {
		idS, err := internal.ParseUint[uint64](serverID)
		if err != nil {
			logger.Errorf("failed to convert server id to int: %v", err)

			return nil, nil
		}

		server, err = internal.GetServerByID(db, int64(idS))
		if err != nil {
			logger.Errorf("failed to get server by id: %v", err)

			return nil, nil
		}

		idA, err := internal.ParseUint[uint64](accountID)
		if err != nil {
			logger.Errorf("failed to convert account id to int: %v", err)

			return nil, nil
		}

		account, err = internal.GetServerAccountByID(db, int64(idA))
		if err != nil {
			logger.Errorf("failed to get account by id: %v", err)

			return nil, nil
		}
	}

	return server, account
}

func localAccountAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		//nolint:forcetypeassert //assertion always succeeds
		tTranslated := pageTranslated("local_account_authentication_page", userLanguage.(string))

		serverID := r.URL.Query().Get("serverID")
		accountID := r.URL.Query().Get("accountID")
		server, account := getServerAndAccount(db, serverID, accountID, logger)

		credentials, filter, credentialAccountFound := listCredentialLocalAccount(server.Name, account.Login, db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "local_account_authentication_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsLocalAccountAuthentication(logger, db, w, r, server, account)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		listSupportedProtocol := supportedProtocolLocalAccount(server.Protocol)
		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := localAccountAuthenticationTemplate.ExecuteTemplate(w, "local_account_authentication_page",
			map[string]any{
				"myPermission":           myPermission,
				"tab":                    tTranslated,
				"username":               user.Username,
				"language":               userLanguage,
				"server":                 server,
				"account":                account,
				"accountCredentials":     credentials,
				"listSupportedProtocol":  listSupportedProtocol,
				"filter":                 filter,
				"currentPage":            currentPage,
				"credentialAccountFound": credentialAccountFound,
				"errMsg":                 errMsg,
				"modalOpen":              modalOpen,
				"modalElement":           modalElement,
				"hasServerID":            true,
				"hasAccountID":           true,
				"sidebarSection":         "connection",
				"sidebarLink":            "server_management",
			}); tmplErr != nil {
			logger.Errorf("render local_account_authentication_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
