package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:dupl // it is not the same function, the calls are different
func listCredentialPartner(partnerName string, db *database.DB, r *http.Request) (
	[]*model.Credential, *Filters, string,
) {
	credentialPartnerFound := ""
	defaultFilter := &Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "partner_authentication_page"); ok {
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

	partnersCredentials, err := internal.ListPartnerCredentials(db, partnerName, "name", true, 0, 0)
	if err != nil {
		return nil, nil, credentialPartnerFound
	}

	if search := urlParams.Get("search"); search != "" && searchCredentialPartner(search, partnersCredentials) == nil {
		credentialPartnerFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialPartnerFound = True

		return []*model.Credential{searchCredentialPartner(search, partnersCredentials)}, filter, credentialPartnerFound
	}

	paginationPage(filter, uint64(len(partnersCredentials)), r)

	partnersCredentialsList, err := internal.ListPartnerCredentials(db, partnerName, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, nil, credentialPartnerFound
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

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
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

	id, err := internal.ParseUint[uint64](credentialPartnerID)
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
	case auth.Password:
		editCredentialPartner.Value = r.FormValue("editCredentialValue")
	case auth.TLSTrustedCertificate, sftp.AuthSSHPublicKey:
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
	case auth.Password:
		newCredentialPartner.Value = r.FormValue("addCredentialValue")
	case auth.TLSTrustedCertificate, sftp.AuthSSHPublicKey:
		newCredentialPartner.Value = r.FormValue("addCredentialValueFile")
	}

	partner, err := internal.GetPartner(db, partnerName)
	if err != nil {
		return fmt.Errorf("failed to get partner: %w", err)
	}

	partner.SetCredOwner(&newCredentialPartner)

	if addErr := internal.InsertCredential(db, &newCredentialPartner); addErr != nil {
		return fmt.Errorf("failed to add credential partner: %w", addErr)
	}

	return nil
}

//nolint:dupl // it is not the same function, the calls are different
func deleteCredentialPartner(partnerName string, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialPartnerID := r.FormValue("deleteCredentialPartner")

	id, err := internal.ParseUint[uint64](credentialPartnerID)
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

//nolint:dupl // method for partner authentication
func callMethodsPartnerAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	partner *model.RemoteAgent,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("deleteCredentialPartner") != "" {
		deleteCredentialPartnerErr := deleteCredentialPartner(partner.Name, db, r)
		if deleteCredentialPartnerErr != nil {
			logger.Errorf("failed to delete credential partner: %v", deleteCredentialPartnerErr)

			return false, deleteCredentialPartnerErr.Error(), "", nil
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("addCredentialPartnerName") != "" {
		addCredentialPartnerErr := addCredentialPartner(partner.Name, db, r)
		if addCredentialPartnerErr != nil {
			logger.Errorf("failed to add partner: %v", addCredentialPartnerErr)
			modalElement = getFormValues(r)

			return false, addCredentialPartnerErr.Error(), "addCredentialPartnerModal", modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editCredentialPartnerID") != "" {
		idEdit := r.FormValue("editCredentialPartnerID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		editredentialPartnerErr := editCredentialPartner(partner.Name, db, r)
		if editredentialPartnerErr != nil {
			logger.Errorf("failed to edit credential partner: %v", editredentialPartnerErr)
			modalElement = getFormValues(r)

			return false, editredentialPartnerErr.Error(), fmt.Sprintf("editCredentialInternalModal_%d", id), modalElement
		}

		http.Redirect(w, r, fmt.Sprintf("%s?partnerID=%d", r.URL.Path, partner.ID), http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func partnerAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		userLanguage := locale.GetLanguage(r)
		tabTrans := pageTranslated("partner_authentication_page", userLanguage)
		myPermission := model.MaskToPerms(user.Permissions)

		var partner *model.RemoteAgent
		var id uint64

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			var err error
			if id, err = internal.ParseUint[uint64](partnerID); err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			if partner, err = internal.GetPartnerByID(db, int64(id)); err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		partnersCredentials, filter, credentialPartnerFound := listCredentialPartner(partner.Name, db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "partner_authentication_page", filter)

		value, errMsg, modalOpen, modalElement := callMethodsPartnerAuthentication(logger, db, w, r, partner)
		if value {
			return
		}

		listSupportedProtocol := supportedProtocolPartner(partner.Protocol)
		currentPage := filter.Offset + 1

		if tmplErr := partnerAuthenticationTemplate.ExecuteTemplate(w, "partner_authentication_page", map[string]any{
			"appName":                constants.AppName,
			"version":                version.Num,
			"compileDate":            version.Date,
			"revision":               version.Commit,
			"docLink":                constants.DocLink(userLanguage),
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
			"modalElement":           modalElement,
			"hasPartnerID":           true,
			"sidebarSection":         "connection",
			"sidebarLink":            "partner_management",
		}); tmplErr != nil {
			logger.Errorf("render partner_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
