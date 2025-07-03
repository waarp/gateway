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

func supportedProtocol(protocol string) []string {
	var listSupportedProtocol []string
	if protocol == "r66" || protocol == "r66-tls" || protocol == "http" || protocol == "https" ||
		protocol == "sftp" || protocol == "pesit" || protocol == "pesit-tls" {
		listSupportedProtocol = append(listSupportedProtocol, "password")
	}

	if protocol == "https" || protocol == "r66-tls" || protocol == "pesit-tls" {
		listSupportedProtocol = append(listSupportedProtocol, "trusted_tls_certificate")
	}

	if protocol == "sftp" {
		listSupportedProtocol = append(listSupportedProtocol, "ssh_public_key")
	}

	if protocol == "r66-tls" {
		listSupportedProtocol = append(listSupportedProtocol, "r66_legacy_certificate")
	}

	return listSupportedProtocol
}

//nolint:dupl // it is not the same function, the calls are different
func listCredentialPartner(partnerName string, db *database.DB, r *http.Request) (
	[]*model.Credential, FiltersPagination, string,
) {
	credentialPartnerFound := ""
	const limit = 30
	filter := FiltersPagination{
		Offset:          0,
		Limit:           limit,
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

	partnersCredentials, err := internal.ListPartnerCredentials(db, partnerName, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, credentialPartnerFound
	}

	if search := r.URL.Query().Get("search"); search != "" && searchCredentialPartner(search, partnersCredentials) == nil {
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

func autocompletionCredentialsPartnersFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")
		var err error
		var partner *model.RemoteAgent
		var id int

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			id, err = strconv.Atoi(partnerID)
			if err != nil {
				http.Error(w, "failed to convert id to int", http.StatusInternalServerError)
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				http.Error(w, "failed to get id", http.StatusInternalServerError)
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

func editCredentialPartner(partnerName string, db *database.DB, r *http.Request) error {
	credentialPartnerID := r.URL.Query().Get("editCredentialPartnerID")

	id, err := strconv.Atoi(credentialPartnerID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editCredentialPartner, err := internal.GetPartnerCredentialByID(db, partnerName, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get credential partner: %w", err)
	}

	if editCredentialPartnerName := r.URL.Query().Get("editCredentialPartnerName"); editCredentialPartnerName != "" {
		editCredentialPartner.Name = editCredentialPartnerName
	}

	if editCredentialPartnerType := r.URL.Query().Get("editCredentialPartnerType"); editCredentialPartnerType != "" {
		editCredentialPartner.Type = editCredentialPartnerType
	}

	if editCredentialPartner.Type == "password" {
		editCredentialPartner.Value = r.URL.Query().Get("editCredentialPartnerValue")
	} else if editCredentialPartner.Type == "trusted_tls_certificate" || editCredentialPartner.Type == "ssh_public_key" {
		editCredentialPartner.Value = r.URL.Query().Get("editCredentialPartnerValue")
	}

	if err = internal.UpdateCredential(db, editCredentialPartner); err != nil {
		return fmt.Errorf("failed to edit credential partner: %w", err)
	}

	return nil
}

func addCredentialPartner(partnerName string, db *database.DB, r *http.Request) error {
	var newCredentialPartner model.Credential

	if newCredentialPartnerName := r.URL.Query().Get("addCredentialPartnerName"); newCredentialPartnerName != "" {
		newCredentialPartner.Name = newCredentialPartnerName
	}

	if newCredentialPartnerType := r.URL.Query().Get("addCredentialPartnerType"); newCredentialPartnerType != "" {
		newCredentialPartner.Type = newCredentialPartnerType
	}

	if newCredentialPartner.Type == "password" {
		newCredentialPartner.Value = r.URL.Query().Get("addCredentialPartnerValue")
	} else if newCredentialPartner.Type == "trusted_tls_certificate" || newCredentialPartner.Type == "ssh_public_key" {
		newCredentialPartner.Value = r.URL.Query().Get("addCredentialPartnerValueFile")
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
	credentialPartnerID := r.URL.Query().Get("deleteCredentialPartner")

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

func callMethods(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	idPartner int, partner *model.RemoteAgent,
) {
	if r.Method == http.MethodGet && r.URL.Query().Get("deleteCredentialPartner") != "" {
		deleteCredentialPartnerErr := deleteCredentialPartner(partner.Name, db, r)
		if deleteCredentialPartnerErr != nil {
			logger.Error("failed to delete credential partner: %v", deleteCredentialPartnerErr)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, idPartner), http.StatusSeeOther)

		return
	}

	if r.Method == http.MethodGet && r.URL.Query().Get("addCredentialPartnerName") != "" {
		addCredentialPartnerErr := addCredentialPartner(partner.Name, db, r)
		if addCredentialPartnerErr != nil {
			logger.Error("failed to delete partner: %v", addCredentialPartnerErr)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, idPartner), http.StatusSeeOther)

		return
	}

	if r.Method == http.MethodGet && r.URL.Query().Get("editCredentialPartnerID") != "" {
		editredentialPartnerErr := editCredentialPartner(partner.Name, db, r)
		if editredentialPartnerErr != nil {
			logger.Error("failed to edit credential partner: %v", editredentialPartnerErr)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, idPartner), http.StatusSeeOther)

		return
	}
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

		callMethods(logger, db, w, r, id, partner)

		listSupportedProtocol := supportedProtocol(partner.Protocol)
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
			"hasPartnerID":           true,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
