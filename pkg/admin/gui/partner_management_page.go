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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

//nolint:dupl // is not similar, is method for partner
func editPartner(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	partnerID := r.FormValue("editPartnerID")

	id, err := strconv.ParseUint(partnerID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editPartner, err := internal.GetPartnerByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	if editPartnerName := r.FormValue("editPartnerName"); editPartnerName != "" {
		editPartner.Name = editPartnerName
	}

	if editPartnerProtocol := r.FormValue("editPartnerProtocol"); editPartnerProtocol != "" {
		editPartner.Protocol = editPartnerProtocol
	}

	if editPartnerHost := r.FormValue("editPartnerHost"); editPartnerHost != "" {
		editPartner.Address.Host = editPartnerHost
	}

	if editPartnerPort := r.FormValue("editPartnerPort"); editPartnerPort != "" {
		var port uint64

		port, err = strconv.ParseUint(editPartnerPort, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		editPartner.Address.Port = uint16(port)
	}

	switch editPartner.Protocol {
	case r66.R66, r66.R66TLS:
		editPartner.ProtoConfig = protoConfigR66Partner(r, editPartner.Protocol)
	case httpconst.HTTP, httpconst.HTTPS:
		editPartner.ProtoConfig = protoConfigHTTPpartner(r, editPartner.Protocol)
	case sftp.SFTP:
		editPartner.ProtoConfig = protoConfigSFTPpartner(r)
	case ftp.FTP, ftp.FTPS:
		editPartner.ProtoConfig = protoConfigFTPpartner(r, editPartner.Protocol)
	case pesit.Pesit, pesit.PesitTLS:
		editPartner.ProtoConfig = protoConfigPeSITPartner(r, editPartner.Protocol)
	}

	if err = internal.UpdatePartner(db, editPartner); err != nil {
		return fmt.Errorf("failed to edit partner: %w", err)
	}

	return nil
}

//nolint:dupl // is not similar, is method for partner
func addPartner(db *database.DB, r *http.Request) error {
	var newPartner model.RemoteAgent

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newPartnerName := r.FormValue("addPartnerName"); newPartnerName != "" {
		newPartner.Name = newPartnerName
	}

	if newPartnerProtocol := r.FormValue("addPartnerProtocol"); newPartnerProtocol != "" {
		newPartner.Protocol = newPartnerProtocol
	}

	if newPartnerHost := r.FormValue("addPartnerHost"); newPartnerHost != "" {
		newPartner.Address.Host = newPartnerHost
	}

	if newPartnerPort := r.FormValue("addPartnerPort"); newPartnerPort != "" {
		port, err := strconv.ParseUint(newPartnerPort, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		newPartner.Address.Port = uint16(port)
	}

	switch newPartner.Protocol {
	case r66.R66, r66.R66TLS:
		newPartner.ProtoConfig = protoConfigR66Partner(r, newPartner.Protocol)
	case httpconst.HTTP, httpconst.HTTPS:
		newPartner.ProtoConfig = protoConfigHTTPpartner(r, newPartner.Protocol)
	case sftp.SFTP:
		newPartner.ProtoConfig = protoConfigSFTPpartner(r)
	case ftp.FTP, ftp.FTPS:
		newPartner.ProtoConfig = protoConfigFTPpartner(r, newPartner.Protocol)
	case pesit.Pesit, pesit.PesitTLS:
		newPartner.ProtoConfig = protoConfigPeSITPartner(r, newPartner.Protocol)
	}

	if err := internal.InsertPartner(db, &newPartner); err != nil {
		return fmt.Errorf("failed to add partner: %w", err)
	}

	return nil
}

func ListPartner(db *database.DB, r *http.Request) ([]*model.RemoteAgent, Filters, string) {
	partnerFound := ""
	defaultFilter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "partner_management_page"); ok {
		filter = saved
	}

	isApply := r.URL.Query().Get("applyFilters") == True
	if isApply {
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

	partner, err := internal.ListPartners(db, "name", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, Filters{}, partnerFound
	}

	if search := urlParams.Get("search"); search != "" && searchPartner(search, partner) == nil {
		partnerFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		partnerFound = True

		return []*model.RemoteAgent{searchPartner(search, partner)}, filter, partnerFound
	}

	filtersPtr, filterProtocol := checkProtocolsFilter(r, isApply, &filter)
	paginationPage(&filter, uint64(len(partner)), r)

	if len(filterProtocol) > 0 {
		var partners []*model.RemoteAgent
		if partners, err = internal.ListPartners(db, "name", filter.OrderAsc, int(filter.Limit),
			int(filter.Offset*filter.Limit), filterProtocol...); err == nil {
			return partners, *filtersPtr, partnerFound
		}
	}

	partners, err := internal.ListPartners(db, "name", filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, partnerFound
	}

	return partners, *filtersPtr, partnerFound
}

//nolint:dupl // is not the same, GetPartnersLike is called
func autocompletionPartnersFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")

		partners, err := internal.GetPartnersLike(db, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(partners))
		for i, u := range partners {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchPartner(partnerNameSearch string, listPartnerSearch []*model.RemoteAgent) *model.RemoteAgent {
	for _, p := range listPartnerSearch {
		if p.Name == partnerNameSearch {
			return p
		}
	}

	return nil
}

//nolint:dupl // is method for partner (different from user method)
func deletePartner(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	partnerID := r.FormValue("deletePartner")

	id, err := strconv.ParseUint(partnerID, 10, 64)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	partner, err := internal.GetPartnerByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeletePartner(db, partner); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

//nolint:dupl // no similar func
func callMethodsPartnerManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("addPartnerName") != "" {
		if newPartnerErr := addPartner(db, r); newPartnerErr != nil {
			logger.Errorf("failed to add partner: %v", newPartnerErr)
			modalElement = getFormValues(r)

			return false, newPartnerErr.Error(), "addPartnerModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("deletePartner") != "" {
		deletePartnerErr := deletePartner(db, r)
		if deletePartnerErr != nil {
			logger.Errorf("failed to delete partner: %v", deletePartnerErr)

			return false, deletePartnerErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editPartnerID") != "" {
		idEdit := r.FormValue("editPartnerID")

		id, err := strconv.ParseUint(idEdit, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		if editPartnerErr := editPartner(db, r); editPartnerErr != nil {
			logger.Errorf("failed to edit partner: %v", editPartnerErr)
			modalElement = getFormValues(r)

			return false, editPartnerErr.Error(), fmt.Sprintf("editPartnerModal_%d", id), modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func partnerManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := pageTranslated("partner_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		partnerList, filter, partnerFound := ListPartner(db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "partner_management_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsPartnerManagement(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := partnerManagementTemplate.ExecuteTemplate(w, "partner_management_page", map[string]any{
			"myPermission":           myPermission,
			"tab":                    tTranslated,
			"username":               user.Username,
			"language":               userLanguage,
			"partner":                partnerList,
			"partnerFound":           partnerFound,
			"filter":                 filter,
			"currentPage":            currentPage,
			"TLSVersions":            TLSVersions,
			"CompatibilityModePeSIT": CompatibilityModePeSIT,
			"KeyExchanges":           sftp.ValidKeyExchanges,
			"Ciphers":                sftp.ValidCiphers,
			"MACs":                   sftp.ValidMACs,
			"protocolsList":          ProtocolsList,
			"errMsg":                 errMsg,
			"modalOpen":              modalOpen,
			"modalElement":           modalElement,
		}); tmplErr != nil {
			logger.Errorf("render partner_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
