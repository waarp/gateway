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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

func editPartner(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	partnerID := r.FormValue("editPartnerID")

	id, err := strconv.Atoi(partnerID)
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
		var port int

		port, err = strconv.Atoi(editPartnerPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		editPartner.Address.Port = uint16(port)
	}

	switch editPartner.Protocol {
	case "r66", "r66-tls":
		editPartner.ProtoConfig = protoConfigR66(r)
	case "sftp":
		editPartner.ProtoConfig = protoConfigSFTP(r)
	case "ftp", "ftps":
		editPartner.ProtoConfig = protoConfigFTP(r, editPartner.Protocol)
	case "pesit", "pesit-tls":
		editPartner.ProtoConfig = protoConfigPeSIT(r, editPartner.Protocol)
	}

	if err = internal.UpdatePartner(db, editPartner); err != nil {
		return fmt.Errorf("failed to edit partner: %w", err)
	}

	return nil
}

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
		port, err := strconv.Atoi(newPartnerPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		newPartner.Address.Port = uint16(port)
	}

	switch newPartner.Protocol {
	case "r66", "r66-tls":
		newPartner.ProtoConfig = protoConfigR66(r)
	case "sftp":
		newPartner.ProtoConfig = protoConfigSFTP(r)
	case "ftp", "ftps":
		newPartner.ProtoConfig = protoConfigFTP(r, newPartner.Protocol)
	case "pesit", "pesit-tls":
		newPartner.ProtoConfig = protoConfigPeSIT(r, newPartner.Protocol)
	}

	if err := internal.InsertPartner(db, &newPartner); err != nil {
		return fmt.Errorf("failed to add partner: %w", err)
	}

	return nil
}

func ListPartner(db *database.DB, r *http.Request) ([]*model.RemoteAgent, FiltersPagination, string) {
	partnerFound := ""
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

	partner, err := internal.ListPartners(db, "name", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, partnerFound
	}

	if search := urlParams.Get("search"); search != "" && searchPartner(search, partner) == nil {
		partnerFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		partnerFound = "true"

		return []*model.RemoteAgent{searchPartner(search, partner)}, filter, partnerFound
	}

	filtersPtr, filterProtocol := protocolsFilter(r, &filter)
	paginationPage(&filter, len(partner), r)

	if len(filterProtocol) > 0 {
		var partners []*model.RemoteAgent
		if partners, err = internal.ListPartners(db, "name", filter.OrderAsc, filter.Limit,
			filter.Offset*filter.Limit, filterProtocol...); err == nil {
			return partners, *filtersPtr, partnerFound
		}
	}

	partners, err := internal.ListPartners(db, "name", filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, partnerFound
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

		if err := json.NewEncoder(w).Encode(names); err != nil {
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

	id, err := strconv.Atoi(partnerID)
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
) (bool, string, string) {
	if r.Method == http.MethodPost && r.FormValue("addPartnerName") != "" {
		if newPartnerErr := addPartner(db, r); newPartnerErr != nil {
			logger.Error("failed to add partner: %v", newPartnerErr)

			return false, newPartnerErr.Error(), "addPartnerModal"
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deletePartner") != "" {
		deletePartnerErr := deletePartner(db, r)
		if deletePartnerErr != nil {
			logger.Error("failed to delete partner: %v", deletePartnerErr)

			return false, deletePartnerErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editPartnerID") != "" {
		idEdit := r.FormValue("editPartnerID")

		id, err := strconv.Atoi(idEdit)
		if err != nil {
			logger.Error("failed to convert id to int: %v", err)

			return false, "", ""
		}

		if editPartnerErr := editPartner(db, r); editPartnerErr != nil {
			logger.Error("failed to edit partner: %v", editPartnerErr)

			return false, editPartnerErr.Error(), fmt.Sprintf("editPartnerModal_%d", id)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func partnerManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := pageTranslated("partner_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		partnerList, filter, partnerFound := ListPartner(db, r)

		value, errMsg, modalOpen := callMethodsPartnerManagement(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if err := partnerManagementTemplate.ExecuteTemplate(w, "partner_management_page", map[string]any{
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
			"errMsg":                 errMsg,
			"modalOpen":              modalOpen,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
