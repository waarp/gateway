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
func listCredentialPartner(partnerName string, db *database.DB, r *http.Request) (
	[]*model.Credential, FiltersPagination, string,
) {
	credentialPartnerFound := ""
	filter := FiltersPagination{
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

	partnersCredentials, err := internal.ListPartnerCredentials(db, partnerName, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, credentialPartnerFound
	}

	if search := urlParams.Get("search"); search != "" && searchCredentialPartner(search, partnersCredentials) == nil {
		credentialPartnerFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialPartnerFound = "true"

		return []*model.Credential{searchCredentialPartner(search, partnersCredentials)}, filter, credentialPartnerFound
	}

	paginationPage(&filter, len(partnersCredentials), r)

	partnersCredentialsList, err := internal.ListPartnerCredentials(db, partnerName, "name",
		filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, credentialPartnerFound
	}

	return partnersCredentialsList, filter, credentialPartnerFound
}

//nolint:dupl // is not the same, GetPartnerAccountByID is called
func autocompletionCredentialsPartnersFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		prefix := urlParams.Get("q")
		var err error
		var partner *model.RemoteAgent
		var id int

		partnerID := urlParams.Get("partnerID")
		if partnerID != "" {
			id, err = strconv.Atoi(partnerID)
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

		credentialsPartners, err := internal.GetCredentialsLike(db, partner, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(credentialsPartners))
		for i, u := range credentialsPartners {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(names); err != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchCredentialPartner(credentialPartnerNameSearch string,
	listCredentialPartnerSearch []*model.Credential,
) *model.Credential {
	for _, cp := range listCredentialPartnerSearch {
		if cp.Name == credentialPartnerNameSearch {
			return cp
		}
	}

	return nil
}

//nolint:dupl // no similar func (is for partner)
func editCredentialPartner(partnerName string, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialPartnerID := r.FormValue("editCredentialPartnerID")

	id, err := strconv.Atoi(credentialPartnerID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editCredentialPartner, err := internal.GetPartnerCredentialByID(db, partnerName, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get credential partner: %w", err)
	}

	if editCredentialPartnerName := r.FormValue("editCredentialPartnerName"); editCredentialPartnerName != "" {
		editCredentialPartner.Name = editCredentialPartnerName
	}

	if editCredentialPartnerType := r.FormValue("editCredentialPartnerType"); editCredentialPartnerType != "" {
		editCredentialPartner.Type = editCredentialPartnerType
	}

	switch editCredentialPartner.Type {
	case "password":
		editCredentialPartner.Value = r.FormValue("editCredentialValue")
	case "trusted_tls_certificate", "ssh_public_key":
		editCredentialPartner.Value = r.FormValue("editCredentialValueFile")
	}

	if err = internal.UpdateCredential(db, editCredentialPartner); err != nil {
		return fmt.Errorf("failed to edit credential partner: %w", err)
	}

	return nil
}

func addCredentialPartner(partnerName string, db *database.DB, r *http.Request) error {
	var newCredentialPartner model.Credential

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newCredentialPartnerName := r.FormValue("addCredentialPartnerName"); newCredentialPartnerName != "" {
		newCredentialPartner.Name = newCredentialPartnerName
	}

	if newCredentialPartnerType := r.FormValue("addCredentialPartnerType"); newCredentialPartnerType != "" {
		newCredentialPartner.Type = newCredentialPartnerType
	}

	switch newCredentialPartner.Type {
	case "password":
		newCredentialPartner.Value = r.FormValue("addCredentialValue")
	case "trusted_tls_certificate", "ssh_public_key":
		newCredentialPartner.Value = r.FormValue("addCredentialValueFile")
	}

	partner, err := internal.GetPartner(db, partnerName)
	if err != nil {
		return fmt.Errorf("failed to get partner: %w", err)
	}

	partner.SetCredOwner(&newCredentialPartner)

	if err := internal.InsertCredential(db, &newCredentialPartner); err != nil {
		return fmt.Errorf("failed to add credential partner: %w", err)
	}

	return nil
}

//nolint:dupl // it is not the same function, the calls are different
func deleteCredentialPartner(partnerName string, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialPartnerID := r.FormValue("deleteCredentialPartner")

	id, err := strconv.Atoi(credentialPartnerID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	credentialPartner, err := internal.GetPartnerCredentialByID(db, partnerName, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteCredential(db, credentialPartner); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func callMethodsPartnerAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	partner *model.RemoteAgent,
) (bool, string, string) {
	if r.Method == http.MethodPost && r.FormValue("deleteCredentialPartner") != "" {
		deleteCredentialPartnerErr := deleteCredentialPartner(partner.Name, db, r)
		if deleteCredentialPartnerErr != nil {
			logger.Error("failed to delete credential partner: %v", deleteCredentialPartnerErr)

			return false, deleteCredentialPartnerErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("addCredentialPartnerName") != "" {
		addCredentialPartnerErr := addCredentialPartner(partner.Name, db, r)
		if addCredentialPartnerErr != nil {
			logger.Error("failed to add partner: %v", addCredentialPartnerErr)

			return false, addCredentialPartnerErr.Error(), "addCredentialPartnerModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editCredentialPartnerID") != "" {
		idEdit := r.FormValue("editCredentialPartnerID")

		id, err := strconv.Atoi(idEdit)
		if err != nil {
			logger.Error("failed to convert id to int: %v", err)

			return false, "", ""
		}

		editredentialPartnerErr := editCredentialPartner(partner.Name, db, r)
		if editredentialPartnerErr != nil {
			logger.Error("failed to edit credential partner: %v", editredentialPartnerErr)

			return false, editredentialPartnerErr.Error(), fmt.Sprintf("editCredentialInternalModal_%d", id)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func partnerAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTrans := pageTranslated("partner_authentication_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var partner *model.RemoteAgent
		var id int

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			id, err = strconv.Atoi(partnerID)
			if err != nil {
				logger.Error("failed to convert id to int: %v", err)
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				logger.Error("failed to get id: %v", err)
			}
		}

		partnersCredentials, filter, credentialPartnerFound := listCredentialPartner(partner.Name, db, r)

		value, errMsg, modalOpen := callMethodsPartnerAuthentication(logger, db, w, r, partner)
		if value {
			return
		}

		listSupportedProtocol := supportedProtocolInternal(partner.Protocol)
		currentPage := filter.Offset + 1

		if err := partnerAuthenticationTemplate.ExecuteTemplate(w, "partner_authentication_page", map[string]any{
			"myPermission":           myPermission,
			"tab":                    tabTrans,
			"username":               user.Username,
			"language":               userLanguage,
			"partner":                partner,
			"partnersCredentials":    partnersCredentials,
			"listSupportedProtocol":  listSupportedProtocol,
			"filter":                 filter,
			"currentPage":            currentPage,
			"credentialPartnerFound": credentialPartnerFound,
			"errMsg":                 errMsg,
			"modalOpen":              modalOpen,
			"hasPartnerID":           true,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
