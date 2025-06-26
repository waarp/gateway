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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type Protocols struct {
	R66      string
	R66TLS   string
	SFTP     string
	HTTP     string
	HTTPS    string
	FTP      string
	FTPS     string
	PeSIT    string
	PeSITTLS string
}

type FiltersPartner struct {
	Offset          int
	Limit           int
	OrderAsc        bool
	DisableNext     bool
	DisablePrevious bool
	Protocols       Protocols
}

//nolint:gochecknoglobals // Constant
var (
	TLSVersions            = []string{protoutils.TLSv10, protoutils.TLSv11, protoutils.TLSv12, protoutils.TLSv13}
	CompatibilityModePeSIT = []string{pesit.CompatibilityModeAxway, pesit.CompatibilityModeNone}
)

func editPartner(db *database.DB, r *http.Request) error {
	partnerID := r.URL.Query().Get("editPartnerID")

	id, err := strconv.Atoi(partnerID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editPartner, err := internal.GetPartnerByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	if editPartnerName := r.URL.Query().Get("editPartnerName"); editPartnerName != "" {
		editPartner.Name = editPartnerName
	}

	if editPartnerProtocol := r.URL.Query().Get("editPartnerProtocol"); editPartnerProtocol != "" {
		editPartner.Protocol = editPartnerProtocol
	}

	if editPartnerHost := r.URL.Query().Get("editPartnerHost"); editPartnerHost != "" {
		editPartner.Address.Host = editPartnerHost
	}

	if editPartnerPort := r.URL.Query().Get("editPartnerPort"); editPartnerPort != "" {
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

func protoConfigR66(r *http.Request) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if serverLogin := r.URL.Query().Get("partnerR66serverLogin"); serverLogin != "" {
		r66ProtoConfig["serverLogin"] = serverLogin
	}

	if blockSize := r.URL.Query().Get("partnerR66blockSize"); blockSize != "" {
		size, err := strconv.Atoi(blockSize)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = uint32(size)
	}

	if noFinalHash := r.URL.Query().Get("noFinalHash"); noFinalHash == "true" {
		r66ProtoConfig["noFinalHash"] = true
	} else {
		r66ProtoConfig["noFinalHash"] = false
	}

	if checkBlockHash := r.URL.Query().Get("checkBlockHash"); checkBlockHash == "true" {
		r66ProtoConfig["checkBlockHash"] = true
	} else {
		r66ProtoConfig["checkBlockHash"] = false
	}

	return r66ProtoConfig
}

func protoConfigSFTP(r *http.Request) map[string]any {
	sftpProtoConfig := make(map[string]any)

	if keyExchanges := r.URL.Query()["keyExchanges[]"]; len(keyExchanges) > 0 {
		sftpProtoConfig["keyExchanges"] = keyExchanges
	}

	if ciphers := r.URL.Query()["ciphers[]"]; len(ciphers) > 0 {
		sftpProtoConfig["ciphers"] = ciphers
	}

	if macs := r.URL.Query()["macs[]"]; len(macs) > 0 {
		sftpProtoConfig["macs"] = macs
	}

	if useStat := r.URL.Query().Get("useStat"); useStat == "true" {
		sftpProtoConfig["useStat"] = true
	} else {
		sftpProtoConfig["useStat"] = false
	}

	if dCCR := r.URL.Query().Get("disableClientConcurrentReads"); dCCR == "true" {
		sftpProtoConfig["disableClientConcurrentReads"] = true
	} else {
		sftpProtoConfig["disableClientConcurrentReads"] = false
	}

	return sftpProtoConfig
}

func protoConfigFTP(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	if disableActiveMode := r.URL.Query().Get("disableActiveMode"); disableActiveMode == "true" {
		ftpProtoConfig["disableActiveMode"] = true
	} else {
		ftpProtoConfig["disableActiveMode"] = false
	}

	if disableEPSV := r.URL.Query().Get("disableEPSV"); disableEPSV == "true" {
		ftpProtoConfig["disableEPSV"] = true
	} else {
		ftpProtoConfig["disableEPSV"] = false
	}

	if protocol == "ftps" { //nolint:nestif // call ftps
		if useImplicitTLS := r.URL.Query().Get("useImplicitTLS"); useImplicitTLS == "true" {
			ftpProtoConfig["useImplicitTLS"] = true
		} else {
			ftpProtoConfig["useImplicitTLS"] = false
		}

		if minTLSVersion := r.URL.Query().Get("partnerFTPSminTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if disableTLSSessionReuse := r.URL.Query().Get("disableTLSSessionReuse"); disableTLSSessionReuse == "true" {
			ftpProtoConfig["disableTLSSessionReuse"] = true
		} else {
			ftpProtoConfig["disableTLSSessionReuse"] = false
		}
	}

	return ftpProtoConfig
}

//nolint:gocyclo,cyclop,funlen // no split method
func protoConfigPeSIT(r *http.Request, protocol string) map[string]any {
	pesitProtoConfig := make(map[string]any)

	if login := r.URL.Query().Get("partnerPeSITlogin"); login != "" {
		pesitProtoConfig["login"] = login
	}

	if disableRestart := r.URL.Query().Get("disableRestart"); disableRestart == "true" {
		pesitProtoConfig["disableRestart"] = true
	} else {
		pesitProtoConfig["disableRestart"] = false
	}

	if disableCheckpoints := r.URL.Query().Get("disableCheckpoints"); disableCheckpoints == "true" {
		pesitProtoConfig["disableCheckpoints"] = true
	} else {
		pesitProtoConfig["disableCheckpoints"] = false
	}

	if checkpointSize := r.URL.Query().Get("partnerPeSITcheckpointSize"); checkpointSize != "" {
		size, err := strconv.Atoi(checkpointSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = uint32(size)
	}

	if checkpointWindow := r.URL.Query().Get("partnerPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := strconv.Atoi(checkpointWindow)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = uint32(size)
	}

	if useNSDU := r.URL.Query().Get("useNSDU"); useNSDU == "true" {
		pesitProtoConfig["useNSDU"] = true
	} else {
		pesitProtoConfig["useNSDU"] = false
	}

	if compatibilityMode := r.URL.Query().Get("partnerPeSITcompatibilityMode"); compatibilityMode != "" {
		pesitProtoConfig["compatibilityMode"] = compatibilityMode
	}

	if maxMessageSize := r.URL.Query().Get("partnerPeSITmaxMessageSize"); maxMessageSize != "" {
		size, err := strconv.Atoi(maxMessageSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["maxMessageSize"] = uint32(size)
	}

	if disablePreConnection := r.URL.Query().Get("disablePreConnection"); disablePreConnection == "true" {
		pesitProtoConfig["disablePreConnection"] = true
	} else {
		pesitProtoConfig["disablePreConnection"] = false
	}

	if protocol == "pesit-tls" {
		if minTLSVersion := r.URL.Query().Get("partnerPeSIT-TLSminTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return pesitProtoConfig
}

func addPartner(db *database.DB, r *http.Request) error {
	var newPartner model.RemoteAgent

	if newPartnerName := r.URL.Query().Get("addPartnerName"); newPartnerName != "" {
		newPartner.Name = newPartnerName
	}

	if newPartnerProtocol := r.URL.Query().Get("addPartnerProtocol"); newPartnerProtocol != "" {
		newPartner.Protocol = newPartnerProtocol
	}

	if newPartnerHost := r.URL.Query().Get("addPartnerHost"); newPartnerHost != "" {
		newPartner.Address.Host = newPartnerHost
	}

	if newPartnerPort := r.URL.Query().Get("addPartnerPort"); newPartnerPort != "" {
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

func paginationPartnerPage(filter *FiltersPartner, lenPartner int, r *http.Request) {
	if r.URL.Query().Get("previous") == "true" && filter.Offset > 0 {
		filter.Offset--
	}

	if r.URL.Query().Get("next") == "true" {
		if filter.Limit*(filter.Offset+1) <= lenPartner {
			filter.Offset++
		}
	}

	if filter.Offset == 0 {
		filter.DisablePrevious = true
	}

	if (filter.Offset+1)*filter.Limit >= lenPartner {
		filter.DisableNext = true
	}
}

func ListPartner(db *database.DB, r *http.Request) ([]*model.RemoteAgent, FiltersPartner, string) {
	partnerFound := ""
	const limit = 30
	filter := FiltersPartner{
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

	partner, err := internal.ListPartners(db, "name", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, FiltersPartner{}, partnerFound
	}

	if search := r.URL.Query().Get("search"); search != "" && searchPartner(search, partner) == nil {
		partnerFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		partnerFound = "true"

		return []*model.RemoteAgent{searchPartner(search, partner)}, filter, partnerFound
	}

	filtersPtr, filterProtocol := protocolsFilter(r, &filter)
	paginationPartnerPage(&filter, len(partner), r)

	if len(filterProtocol) > 0 {
		var partners []*model.RemoteAgent
		if partners, err = internal.ListPartners(db, "name", filter.OrderAsc, filter.Limit,
			filter.Offset*filter.Limit, filterProtocol...); err == nil {
			return partners, *filtersPtr, partnerFound
		}
	}

	partners, err := internal.ListPartners(db, "name", filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPartner{}, partnerFound
	}

	return partners, *filtersPtr, partnerFound
}

func protocolsFilter(r *http.Request, filter *FiltersPartner) (*FiltersPartner, []string) {
	var filterProtocol []string
	if filter.Protocols.R66 = r.URL.Query().Get("filterProtocolR66"); filter.Protocols.R66 == "true" {
		filterProtocol = append(filterProtocol, "r66")
	}

	if filter.Protocols.R66TLS = r.URL.Query().Get("filterProtocolR66-TLS"); filter.Protocols.R66TLS == "true" {
		filterProtocol = append(filterProtocol, "r66-tls")
	}

	if filter.Protocols.SFTP = r.URL.Query().Get("filterProtocolSFTP"); filter.Protocols.SFTP == "true" {
		filterProtocol = append(filterProtocol, "sftp")
	}

	if filter.Protocols.HTTP = r.URL.Query().Get("filterProtocolHTTP"); filter.Protocols.HTTP == "true" {
		filterProtocol = append(filterProtocol, "http")
	}

	if filter.Protocols.HTTPS = r.URL.Query().Get("filterProtocolHTTPS"); filter.Protocols.HTTPS == "true" {
		filterProtocol = append(filterProtocol, "https")
	}

	if filter.Protocols.FTP = r.URL.Query().Get("filterProtocolFTP"); filter.Protocols.FTP == "true" {
		filterProtocol = append(filterProtocol, "ftp")
	}

	if filter.Protocols.FTPS = r.URL.Query().Get("filterProtocolFTPS"); filter.Protocols.FTPS == "true" {
		filterProtocol = append(filterProtocol, "ftps")
	}

	if filter.Protocols.PeSIT = r.URL.Query().Get("filterProtocolPeSIT"); filter.Protocols.PeSIT == "true" {
		filterProtocol = append(filterProtocol, "pesit")
	}

	if filter.Protocols.PeSITTLS = r.URL.Query().Get("filterProtocolPeSIT-TLS"); filter.Protocols.PeSITTLS == "true" {
		filterProtocol = append(filterProtocol, "pesit-tls")
	}

	return filter, filterProtocol
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

func deletePartner(db *database.DB, r *http.Request) error {
	partnerID := r.URL.Query().Get("deletePartner")

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

func partnerManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := pageTranslated("partner_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		partnerList, filter, partnerFound := ListPartner(db, r)

		if r.Method == http.MethodGet && r.URL.Query().Get("addPartnerName") != "" {
			if newPartnerErr := addPartner(db, r); newPartnerErr != nil {
				logger.Error("failed to add partner: %v", newPartnerErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		if r.Method == http.MethodGet && r.URL.Query().Get("deletePartner") != "" {
			deletePartnerErr := deletePartner(db, r)
			if deletePartnerErr != nil {
				logger.Error("failed to delete partner: %v", deletePartnerErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		if r.Method == http.MethodGet && r.URL.Query().Get("editPartnerID") != "" {
			if editPartnerErr := editPartner(db, r); editPartnerErr != nil {
				logger.Error("failed to edit partner: %v", editPartnerErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

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
			"KeyExchanges":           sftp.ValidKeyExchanges, "Ciphers": sftp.ValidCiphers, "MACs": sftp.ValidMACs,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
