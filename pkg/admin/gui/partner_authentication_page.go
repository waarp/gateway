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

func supportedProtocolInternal(protocol string) []string {
	supportedProtocolsInternal := map[string][]string{
		"r66":       {"password"},
		"r66-tls":   {"password", "trusted_tls_certificate", "r66_legacy_certificate"},
		"http":      {"password"},
		"https":     {"password", "trusted_tls_certificate"},
		"sftp":      {"password", "ssh_public_key"},
		"pesit":     {"password"},
		"pesit-tls": {"password", "trusted_tls_certificate"},
	}

	return supportedProtocolsInternal[protocol]
}

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

	switch editCredentialPartner.Type {
	case "password":
		editCredentialPartner.Value = r.URL.Query().Get("editCredentialValue")
	case "trusted_tls_certificate", "ssh_public_key":
		editCredentialPartner.Value = r.URL.Query().Get("editCredentialValueFile")
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

	switch newCredentialPartner.Type {
	case "password":
		newCredentialPartner.Value = r.URL.Query().Get("addCredentialValue")
	case "trusted_tls_certificate", "ssh_public_key":
		newCredentialPartner.Value = r.URL.Query().Get("addCredentialValueFile")
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

func callMethodsPartnerAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
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
			logger.Error("failed to add partner: %v", addCredentialPartnerErr)
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

		callMethodsPartnerAuthentication(logger, db, w, r, id, partner)

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
			"hasPartnerID":           true,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
