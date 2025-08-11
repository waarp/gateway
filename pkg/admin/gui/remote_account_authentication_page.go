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
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

//nolint:dupl // no similar func (is for remote_account)
func listCredentialRemoteAccount(partnerName, login string, db *database.DB, r *http.Request) (
	[]*model.Credential, Filters, string,
) {
	credentialAccountFound := ""
	filter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	urlParams := r.URL.Query()

	filter.OrderAsc = urlParams.Get("orderAsc") == "true"

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

	accountsCredentials, err := internal.ListPartnerAccountCredentials(db, partnerName, login, "name", true, 0, 0)
	if err != nil {
		return nil, Filters{}, credentialAccountFound
	}

	if search := urlParams.Get("search"); search != "" && searchCredentialRemoteAccount(search,
		accountsCredentials) == nil {
		credentialAccountFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialAccountFound = "true"

		return []*model.Credential{searchCredentialRemoteAccount(search, accountsCredentials)}, filter, credentialAccountFound
	}

	paginationPage(&filter, uint64(len(accountsCredentials)), r)

	accountsCredentialsList, err := internal.ListPartnerAccountCredentials(db, partnerName, login, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, credentialAccountFound
	}

	return accountsCredentialsList, filter, credentialAccountFound
}

//nolint:dupl // is not the same, GetCredentialsLike is called
func autocompletionCredentialsRemoteAccountsFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		prefix := urlParams.Get("q")
		var err error
		var account *model.RemoteAccount
		var idA uint64

		accountID := urlParams.Get("accountID")

		if accountID != "" {
			idA, err = strconv.ParseUint(accountID, 10, 64)
			if err != nil {
				http.Error(w, "failed to convert account id to int", http.StatusInternalServerError)

				return
			}

			account, err = internal.GetPartnerAccountByID(db, int64(idA))
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

func searchCredentialRemoteAccount(credentialAccountNameSearch string,
	listCredentialAccountSearch []*model.Credential,
) *model.Credential {
	for _, cra := range listCredentialAccountSearch {
		if cra.Name == credentialAccountNameSearch {
			return cra
		}
	}

	return nil
}

func addCredentialRemoteAccount(partnerName, login string, db *database.DB, r *http.Request) error {
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
	case sftp.AuthSSHPrivateKey:
		newCredentialAccount.Value = r.FormValue("addCredentialValueFile")
	case auth.TLSCertificate:
		newCredentialAccount.Value = r.FormValue("addCredentialValueFile1")
		newCredentialAccount.Value2 = r.FormValue("addCredentialValueFile2")
	case pesit.PreConnectionAuth:
		newCredentialAccount.Value = r.FormValue("addCredentialValue1")
		newCredentialAccount.Value2 = r.FormValue("addCredentialValue2")
	}

	account, err := internal.GetPartnerAccount(db, partnerName, login)
	if err != nil {
		return fmt.Errorf("failed to get partner account: %w", err)
	}

	account.SetCredOwner(&newCredentialAccount)

	if addErr := internal.InsertCredential(db, &newCredentialAccount); addErr != nil {
		return fmt.Errorf("failed to add credential account: %w", addErr)
	}

	return nil
}

//nolint:dupl // no similar func (is for account)
func editCredentialRemoteAccount(account *model.RemoteAccount, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialAccountID := r.FormValue("editCredentialAccountID")

	id, err := strconv.ParseUint(credentialAccountID, 10, 64)
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
	case sftp.AuthSSHPrivateKey:
		editCredentialAccount.Value = r.FormValue("editCredentialValueFile")
	case auth.TLSCertificate:
		editCredentialAccount.Value = r.FormValue("editCredentialValueFile1")
		editCredentialAccount.Value2 = r.FormValue("editCredentialValueFile2")
	case pesit.PreConnectionAuth:
		editCredentialAccount.Value = r.FormValue("editCredentialValue1")
		editCredentialAccount.Value2 = r.FormValue("editCredentialValue2")
	}

	if err = internal.UpdateCredential(db, editCredentialAccount); err != nil {
		return fmt.Errorf("failed to edit credential account: %w", err)
	}

	return nil
}

func deleteCredentialRemoteAccount(account *model.RemoteAccount, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialAccountID := r.FormValue("deleteCredentialAccount")

	id, err := strconv.ParseUint(credentialAccountID, 10, 64)
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

func callMethodsRemoteAccountAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	partner *model.RemoteAgent, account *model.RemoteAccount,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("deleteCredentialAccount") != "" {
		deleteCredentialAccountErr := deleteCredentialRemoteAccount(account, db, r)
		if deleteCredentialAccountErr != nil {
			logger.Errorf("failed to delete credential account: %v", deleteCredentialAccountErr)

			return false, deleteCredentialAccountErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d&accountID=%d", r.URL.Path, partner.ID, account.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("addCredentialAccountName") != "" {
		addCredentialAccountErr := addCredentialRemoteAccount(partner.Name, account.Login, db, r)
		if addCredentialAccountErr != nil {
			logger.Errorf("failed to add credential account: %v", addCredentialAccountErr)

			return false, addCredentialAccountErr.Error(), "addCredentialAccountModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d&accountID=%d", r.URL.Path, partner.ID, account.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editCredentialAccountID") != "" {
		idEdit := r.FormValue("editCredentialAccountID")

		id, err := strconv.ParseUint(idEdit, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", ""
		}

		editCredentialAccountErr := editCredentialRemoteAccount(account, db, r)
		if editCredentialAccountErr != nil {
			logger.Errorf("failed to edit credential account: %v", editCredentialAccountErr)

			return false, editCredentialAccountErr.Error(), fmt.Sprintf("editCredentialExternalModal_%d", id)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d&accountID=%d", r.URL.Path, partner.ID, account.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func getPartnerAndAccount(db *database.DB, partnerID, accountID string, logger *log.Logger) (
	*model.RemoteAgent, *model.RemoteAccount,
) {
	var partner *model.RemoteAgent
	var account *model.RemoteAccount

	if partnerID != "" && accountID != "" {
		idP, err := strconv.ParseUint(partnerID, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert partner id to int: %v", err)

			return nil, nil
		}

		partner, err = internal.GetPartnerByID(db, int64(idP))
		if err != nil {
			logger.Errorf("failed to get partner by id: %v", err)

			return nil, nil
		}

		idA, err := strconv.ParseUint(accountID, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert account id to int: %v", err)

			return nil, nil
		}

		account, err = internal.GetPartnerAccountByID(db, int64(idA))
		if err != nil {
			logger.Errorf("failed to get account by id: %v", err)

			return nil, nil
		}
	}

	return partner, account
}

func remoteAccountAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("remote_account_authentication_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)

		partnerID := r.URL.Query().Get("partnerID")
		accountID := r.URL.Query().Get("accountID")
		partner, account := getPartnerAndAccount(db, partnerID, accountID, logger)

		credentials, filter, credentialAccountFound := listCredentialRemoteAccount(partner.Name, account.Login, db, r)

		value, errMsg, modalOpen := callMethodsRemoteAccountAuthentication(logger, db, w, r, partner, account)
		if value {
			return
		}

		listSupportedProtocol := supportedProtocolExternal(partner.Protocol)
		currentPage := filter.Offset + 1

		if tmplErr := remoteAccountAuthenticationTemplate.ExecuteTemplate(w, "remote_account_authentication_page",
			map[string]any{
				"myPermission":           myPermission,
				"tab":                    tTranslated,
				"username":               user.Username,
				"language":               userLanguage,
				"partner":                partner,
				"account":                account,
				"accountCredentials":     credentials,
				"listSupportedProtocol":  listSupportedProtocol,
				"filter":                 filter,
				"currentPage":            currentPage,
				"credentialAccountFound": credentialAccountFound,
				"errMsg":                 errMsg,
				"modalOpen":              modalOpen,
				"hasPartnerID":           true,
				"hasAccountID":           true,
			}); tmplErr != nil {
			logger.Errorf("render remote_account_authentication_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
