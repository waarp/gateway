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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

func addServer(db *database.DB, r *http.Request) error {
	var newServer model.LocalAgent

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newServerName := r.FormValue("addServerName"); newServerName != "" {
		newServer.Name = newServerName
	}

	if newServerProtocol := r.FormValue("addServerProtocol"); newServerProtocol != "" {
		newServer.Protocol = newServerProtocol
	}

	if newServerHost := r.FormValue("addServerHost"); newServerHost != "" {
		newServer.Address.Host = newServerHost
	}

	if newServerPort := r.FormValue("addServerPort"); newServerPort != "" {
		port, err := strconv.Atoi(newServerPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		newServer.Address.Port = uint16(port)
	}

	if newServerRootDir := r.FormValue("addServerRootDir"); newServerRootDir != "" {
		newServer.RootDir = newServerRootDir
	}

	if newServerReceiveDir := r.FormValue("addServerReceiveDir"); newServerReceiveDir != "" {
		newServer.ReceiveDir = newServerReceiveDir
	}

	if newServerSendDir := r.FormValue("addServerSendDir"); newServerSendDir != "" {
		newServer.SendDir = newServerSendDir
	}

	if newServerTmpReceiveDir := r.FormValue("addServerTmpReceiveDir"); newServerTmpReceiveDir != "" {
		newServer.TmpReceiveDir = newServerTmpReceiveDir
	}

	switch newServer.Protocol {
	case r66.R66, r66.R66TLS:
		newServer.ProtoConfig = protoConfigR66Server(r)
	case sftp.SFTP:
		newServer.ProtoConfig = protoConfigSFTPServer(r)
	case ftp.FTP, ftp.FTPS:
		newServer.ProtoConfig = protoConfigFTPServer(r, newServer.Protocol)
	case pesit.Pesit, pesit.PesitTLS:
		newServer.ProtoConfig = protoConfigPeSITServer(r, newServer.Protocol)
	}

	if err := internal.InsertServer(db, &newServer); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	return nil
}

//nolint:funlen // unique method
func editServer(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	serverID := r.FormValue("editServerID")

	id, err := strconv.Atoi(serverID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editServer, err := internal.GetServerByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	if editServerName := r.FormValue("editServerName"); editServerName != "" {
		editServer.Name = editServerName
	}

	if editServerProtocol := r.FormValue("editServerProtocol"); editServerProtocol != "" {
		editServer.Protocol = editServerProtocol
	}

	if editServerHost := r.FormValue("editServerHost"); editServerHost != "" {
		editServer.Address.Host = editServerHost
	}

	if editServerPort := r.FormValue("editServerPort"); editServerPort != "" {
		var port int

		port, err = strconv.Atoi(editServerPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		editServer.Address.Port = uint16(port)
	}

	if editServerRootDir := r.FormValue("editServerRootDir"); editServerRootDir != "" {
		editServer.RootDir = editServerRootDir
	}

	if editServerReceiveDir := r.FormValue("editServerReceiveDir"); editServerReceiveDir != "" {
		editServer.ReceiveDir = editServerReceiveDir
	}

	if editServerSendDir := r.FormValue("editServerSendDir"); editServerSendDir != "" {
		editServer.SendDir = editServerSendDir
	}

	if editServerTmpReceiveDir := r.FormValue("editServerTmpReceiveDir"); editServerTmpReceiveDir != "" {
		editServer.TmpReceiveDir = editServerTmpReceiveDir
	}

	switch editServer.Protocol {
	case r66.R66, r66.R66TLS:
		editServer.ProtoConfig = protoConfigR66Server(r)
	case sftp.SFTP:
		editServer.ProtoConfig = protoConfigSFTPServer(r)
	case ftp.FTP, ftp.FTPS:
		editServer.ProtoConfig = protoConfigFTPServer(r, editServer.Protocol)
	case pesit.Pesit, pesit.PesitTLS:
		editServer.ProtoConfig = protoConfigPeSITServer(r, editServer.Protocol)
	}

	if err = internal.UpdateServer(db, editServer); err != nil {
		return fmt.Errorf("failed to edit server: %w", err)
	}

	return nil
}

//nolint:dupl // no similar func
func deleteServer(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	serverID := r.FormValue("deleteServer")

	id, err := strconv.Atoi(serverID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	server, err := internal.GetServerByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteServer(db, server); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func listServer(db *database.DB, r *http.Request) ([]*model.LocalAgent, FiltersPagination, string) {
	serverFound := ""
	filter := FiltersPagination{
		Offset:          0,
		Limit:           DefaultLimitPagination,
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

	server, err := internal.ListServers(db, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, serverFound
	}

	if search := urlParams.Get("search"); search != "" && searchServer(search, server) == nil {
		serverFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		serverFound = "true"

		return []*model.LocalAgent{searchServer(search, server)}, filter, serverFound
	}

	filtersPtr, filterProtocol := protocolsFilter(r, &filter)
	paginationPage(&filter, len(server), r)

	if len(filterProtocol) > 0 {
		var servers []*model.LocalAgent
		if servers, err = internal.ListServers(db, "name", filter.OrderAsc, filter.Limit,
			filter.Offset*filter.Limit, filterProtocol...); err == nil {
			return servers, *filtersPtr, serverFound
		}
	}

	servers, err := internal.ListServers(db, "name", filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, serverFound
	}

	return servers, filter, serverFound
}

func searchServer(serverNameSearch string, listServerSearch []*model.LocalAgent,
) *model.LocalAgent {
	for _, s := range listServerSearch {
		if s.Name == serverNameSearch {
			return s
		}
	}

	return nil
}

//nolint:dupl // no similar func
func autocompletionServersFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")

		servers, err := internal.GetServersLike(db, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(servers))
		for i, u := range servers {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

//nolint:dupl // no similar func
func callMethodsServerManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addServerName") != "" {
		if newServerErr := addServer(db, r); newServerErr != nil {
			logger.Errorf("failed to add server: %v", newServerErr)

			return false, newServerErr.Error(), "addServerModal"
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteServer") != "" {
		deleteServerErr := deleteServer(db, r)
		if deleteServerErr != nil {
			logger.Errorf("failed to delete server: %v", deleteServerErr)

			return false, deleteServerErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editServerID") != "" {
		idEdit := r.FormValue("editServerID")

		id, err := strconv.Atoi(idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", ""
		}

		if editServerErr := editServer(db, r); editServerErr != nil {
			logger.Errorf("failed to edit server: %v", editServerErr)

			return false, editServerErr.Error(), fmt.Sprintf("editServerModal_%d", id)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func serverManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("server_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		serverList, filter, serverFound := listServer(db, r)

		value, errMsg, modalOpen := callMethodsServerManagement(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := serverManagementTemplate.ExecuteTemplate(w, "server_management_page", map[string]any{
			"myPermission":           myPermission,
			"tab":                    tabTranslated,
			"username":               user.Username,
			"language":               userLanguage,
			"server":                 serverList,
			"serverFound":            serverFound,
			"filter":                 filter,
			"currentPage":            currentPage,
			"TLSVersions":            TLSVersions,
			"CompatibilityModePeSIT": CompatibilityModePeSIT,
			"TLSRequirement":         TLSRequirement,
			"KeyExchanges":           sftp.ValidKeyExchanges,
			"Ciphers":                sftp.ValidCiphers,
			"MACs":                   sftp.ValidMACs,
			"errMsg":                 errMsg,
			"modalOpen":              modalOpen,
		}); tmplErr != nil {
			logger.Errorf("render server_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
