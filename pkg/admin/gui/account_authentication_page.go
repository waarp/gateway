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

func supportedProtocolExternal(protocol string) []string {
	var listSupportedProtocol []string
	if protocol == "r66" || protocol == "r66-tls" || protocol == "http" || protocol == "https" ||
		protocol == "sftp" || protocol == "pesit" {
		listSupportedProtocol = append(listSupportedProtocol, "password")
	}

	if protocol == "https" || protocol == "r66-tls" || protocol == "pesit-tls" {
		listSupportedProtocol = append(listSupportedProtocol, "tls_certificate")
	}

	if protocol == "sftp" {
		listSupportedProtocol = append(listSupportedProtocol, "ssh_private_key")
	}

	if protocol == "r66-tls" {
		listSupportedProtocol = append(listSupportedProtocol, "r66_legacy_certificate")
	}

	if protocol == "pesit" || protocol == "pesit-tls" {
		listSupportedProtocol = append(listSupportedProtocol, "pesit_pre-connection_auth")
	}

	return listSupportedProtocol
}

func listCredentialAccount(partnerName, login string, db *database.DB, r *http.Request) (
	[]*model.Credential, FiltersPagination, string,
) {
	credentialAccountFound := ""
	filter := FiltersPagination{
		Offset:          0,
		Limit:           LimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	if r.URL.Query().Get("orderAsc") == "true" {
		filter.OrderAsc = true
	} else if r.URL.Query().Get("orderAsc") == "false" {
		filter.OrderAsc = false
	}

	if limitRes := r.URL.Query().Get("limit"); limitRes != "" {
		if l, err := strconv.Atoi(limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := r.URL.Query().Get("offset"); offsetRes != "" {
		if o, err := strconv.Atoi(offsetRes); err == nil {
			filter.Offset = o
		}
	}

	accountsCredentials, err := internal.ListPartnerAccountCredentials(db, partnerName, login, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, credentialAccountFound
	}

	if search := r.URL.Query().Get("search"); search != "" && searchCredentialAccount(search, accountsCredentials) == nil {
		credentialAccountFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialAccountFound = "true"

		return []*model.Credential{searchCredentialAccount(search, accountsCredentials)}, filter, credentialAccountFound
	}

	paginationPage(&filter, len(accountsCredentials), r)

	accountsCredentialsList, err := internal.ListPartnerAccountCredentials(db, partnerName, login, "name",
		filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, credentialAccountFound
	}

	return accountsCredentialsList, filter, credentialAccountFound
}

func autocompletionCredentialsAccountsFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")
		var err error
		var partner *model.RemoteAgent
		var account *model.RemoteAccount
		var idP, idA int

		partnerID := r.URL.Query().Get("partnerID")
		accountID := r.URL.Query().Get("accountID")

		if partnerID != "" && accountID != "" {
			idP, err = strconv.Atoi(partnerID)
			if err != nil {
				http.Error(w, "failed to convert partner id to int", http.StatusInternalServerError)

				return
			}

			partner, err = internal.GetPartnerByID(db, int64(idP))
			if err != nil {
				http.Error(w, "failed to get partner by id", http.StatusInternalServerError)

				return
			}

			idA, err = strconv.Atoi(accountID)
			if err != nil {
				http.Error(w, "failed to convert account id to int", http.StatusInternalServerError)

				return
			}

			account, err = internal.GetPartnerAccountByID(db, partner.Name, int64(idA))
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

		if err := json.NewEncoder(w).Encode(names); err != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchCredentialAccount(credentialAccountNameSearch string,
	listCredentialAccountSearch []*model.Credential,
) *model.Credential {
	for _, ca := range listCredentialAccountSearch {
		if ca.Name == credentialAccountNameSearch {
			return ca
		}
	}

	return nil
}

func addCredentialAccount(partnerName, login string, db *database.DB, r *http.Request) error {
	var newCredentialAccount model.Credential

	if newCredentialAccountName := r.URL.Query().Get("addCredentialAccountName"); newCredentialAccountName != "" {
		newCredentialAccount.Name = newCredentialAccountName
	}

	if newCredentialAccountType := r.URL.Query().Get("addCredentialAccountType"); newCredentialAccountType != "" {
		newCredentialAccount.Type = newCredentialAccountType
	}

	switch newCredentialAccount.Type {
	case "password":
		newCredentialAccount.Value = r.URL.Query().Get("addCredentialValue")
	case "ssh_private_key":
		newCredentialAccount.Value = r.URL.Query().Get("addCredentialValueFile")
	case "tls_certificate":
		newCredentialAccount.Value = r.URL.Query().Get("addCredentialValueFile1")
		newCredentialAccount.Value2 = r.URL.Query().Get("addCredentialValueFile2")
	case "pesit_pre-connection_auth":
		newCredentialAccount.Value = r.URL.Query().Get("addCredentialValue1")
		newCredentialAccount.Value2 = r.URL.Query().Get("addCredentialValue2")
	}

	account, err := internal.GetPartnerAccount(db, partnerName, login)
	if err != nil {
		return fmt.Errorf("failed to get partner account: %w", err)
	}

	account.SetCredOwner(&newCredentialAccount)

	if err := internal.InsertCredential(db, &newCredentialAccount); err != nil {
		return fmt.Errorf("failed to add credential account: %w", err)
	}

	return nil
}

func editCredentialAccount(account *model.RemoteAccount, db *database.DB, r *http.Request) error {
	credentialAccountID := r.URL.Query().Get("editCredentialAccountID")

	id, err := strconv.Atoi(credentialAccountID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editCredentialAccount, err := internal.GetCredentialByID(db, account, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get credential account: %w", err)
	}

	if editCredentialAccountName := r.URL.Query().Get("editCredentialAccountName"); editCredentialAccountName != "" {
		editCredentialAccount.Name = editCredentialAccountName
	}

	if editCredentialAccountType := r.URL.Query().Get("editCredentialAccountType"); editCredentialAccountType != "" {
		editCredentialAccount.Type = editCredentialAccountType
	}

	switch editCredentialAccount.Type {
	case "password":
		editCredentialAccount.Value = r.URL.Query().Get("editCredentialValue")
	case "ssh_private_key":
		editCredentialAccount.Value = r.URL.Query().Get("editCredentialValueFile")
	case "tls_certificate":
		editCredentialAccount.Value = r.URL.Query().Get("editCredentialValueFile1")
		editCredentialAccount.Value2 = r.URL.Query().Get("editCredentialValueFile2")
	case "pesit_pre-connection_auth":
		editCredentialAccount.Value = r.URL.Query().Get("editCredentialValue1")
		editCredentialAccount.Value2 = r.URL.Query().Get("editCredentialValue2")
	}

	if err = internal.UpdateCredential(db, editCredentialAccount); err != nil {
		return fmt.Errorf("failed to edit credential account: %w", err)
	}

	return nil
}

func deleteCredentialAccount(account *model.RemoteAccount, db *database.DB, r *http.Request) error {
	credentialAccountID := r.URL.Query().Get("deleteCredentialAccount")

	id, err := strconv.Atoi(credentialAccountID)
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

func callMethodsAccountAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	partner *model.RemoteAgent, account *model.RemoteAccount,
) {
	if r.Method == http.MethodGet && r.URL.Query().Get("deleteCredentialAccount") != "" {
		deleteCredentialAccountErr := deleteCredentialAccount(account, db, r)
		if deleteCredentialAccountErr != nil {
			logger.Error("failed to delete credential account: %v", deleteCredentialAccountErr)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d&accountID=%d", r.URL.Path, partner.ID, account.ID),
			http.StatusSeeOther)

		return
	}

	if r.Method == http.MethodGet && r.URL.Query().Get("addCredentialAccountName") != "" {
		addCredentialAccountErr := addCredentialAccount(partner.Name, account.Login, db, r)
		if addCredentialAccountErr != nil {
			logger.Error("failed to add credential account: %v", addCredentialAccountErr)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d&accountID=%d", r.URL.Path, partner.ID, account.ID),
			http.StatusSeeOther)

		return
	}

	if r.Method == http.MethodGet && r.URL.Query().Get("editCredentialAccountName") != "" {
		editCredentialAccountErr := editCredentialAccount(account, db, r)
		if editCredentialAccountErr != nil {
			logger.Error("failed to edit credential account: %v", editCredentialAccountErr)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d&accountID=%d", r.URL.Path, partner.ID, account.ID),
			http.StatusSeeOther)

		return
	}
}

func accountAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("account_authentication_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var partner *model.RemoteAgent
		var account *model.RemoteAccount

		partnerID := r.URL.Query().Get("partnerID")
		accountID := r.URL.Query().Get("accountID")

		if partnerID != "" && accountID != "" {
			idP, err := strconv.Atoi(partnerID)
			if err != nil {
				logger.Error("failed to convert partner id to int: %v", err)
			}

			partner, err = internal.GetPartnerByID(db, int64(idP))
			if err != nil {
				logger.Error("failed to get partner by id: %v", err)
			}

			idA, err := strconv.Atoi(accountID)
			if err != nil {
				logger.Error("failed to convert account id to int: %v", err)
			}

			account, err = internal.GetPartnerAccountByID(db, partner.Name, int64(idA))
			if err != nil {
				logger.Error("failed to get account by id: %v", err)
			}
		}

		credentials, filter, credentialAccountFound := listCredentialAccount(partner.Name, account.Login, db, r)

		callMethodsAccountAuthentication(logger, db, w, r, partner, account)

		listSupportedProtocol := supportedProtocolExternal(partner.Protocol)
		currentPage := filter.Offset + 1

		if err := accountAuthenticationTemplate.ExecuteTemplate(w, "account_authentication_page", map[string]any{
			"myPermission": myPermission,
			"tab":          tTranslated,
			"username":     user.Username,
			"language":     userLanguage,
			"partner":      partner, "account": account,
			"accountCredentials":    credentials,
			"listSupportedProtocol": listSupportedProtocol,
			"filter":                filter, "currentPage": currentPage,
			"credentialAccountFound": credentialAccountFound,
			"hasPartnerID":           true, "hasAccountID": true,
		}); err != nil {
			logger.Error("render account_authentication_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
