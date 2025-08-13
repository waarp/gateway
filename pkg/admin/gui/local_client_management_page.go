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

//nolint:dupl // is not similar, is method for local client
func addLocalClient(db *database.DB, r *http.Request) error {
	var newLocalClient model.Client

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newLocalClientName := r.FormValue("addLocalClientName"); newLocalClientName != "" {
		newLocalClient.Name = newLocalClientName
	}

	if newLocalClientProtocol := r.FormValue("addLocalClientProtocol"); newLocalClientProtocol != "" {
		newLocalClient.Protocol = newLocalClientProtocol
	}

	if newLocalClientHost := r.FormValue("addLocalClientHost"); newLocalClientHost != "" {
		newLocalClient.LocalAddress.Host = newLocalClientHost
	}

	if newLocalClientPort := r.FormValue("addLocalClientPort"); newLocalClientPort != "" {
		port, err := strconv.Atoi(newLocalClientPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		newLocalClient.LocalAddress.Port = uint16(port)
	}

	switch newLocalClient.Protocol {
	case "r66", "r66-tls":
		newLocalClient.ProtoConfig = protoConfigR66Client(r)
	case "sftp":
		newLocalClient.ProtoConfig = protoConfigSFTPClient(r)
	case "ftp", "ftps":
		newLocalClient.ProtoConfig = protoConfigFTPClient(r, newLocalClient.Protocol)
	case "pesit", "pesit-tls":
		newLocalClient.ProtoConfig = protoConfigPeSITClient(r)
	}

	if err := internal.InsertClient(db, &newLocalClient); err != nil {
		return fmt.Errorf("failed to add local client: %w", err)
	}

	return nil
}

//nolint:dupl // is not similar, is method for local client
func editLocalClient(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	localClientID := r.FormValue("editLocalClientID")

	id, err := strconv.Atoi(localClientID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editLocalClient, err := internal.GetClientByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get localClient: %w", err)
	}

	if editLocalClientName := r.FormValue("editLocalClientName"); editLocalClientName != "" {
		editLocalClient.Name = editLocalClientName
	}

	if editLocalClientProtocol := r.FormValue("editLocalClientProtocol"); editLocalClientProtocol != "" {
		editLocalClient.Protocol = editLocalClientProtocol
	}

	if editLocalClientHost := r.FormValue("editLocalClientHost"); editLocalClientHost != "" {
		editLocalClient.LocalAddress.Host = editLocalClientHost
	}

	if editLocalClientPort := r.FormValue("editLocalClientPort"); editLocalClientPort != "" {
		var port int

		port, err = strconv.Atoi(editLocalClientPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		editLocalClient.LocalAddress.Port = uint16(port)
	}

	switch editLocalClient.Protocol {
	case "r66", "r66-tls":
		editLocalClient.ProtoConfig = protoConfigR66Client(r)
	case "sftp":
		editLocalClient.ProtoConfig = protoConfigSFTPClient(r)
	case "ftp", "ftps":
		editLocalClient.ProtoConfig = protoConfigFTPClient(r, editLocalClient.Protocol)
	case "pesit", "pesit-tls":
		editLocalClient.ProtoConfig = protoConfigPeSITClient(r)
	}

	if err = internal.UpdateClient(db, editLocalClient); err != nil {
		return fmt.Errorf("failed to edit localClient: %w", err)
	}

	return nil
}

//nolint:dupl // is not similar, is method for local client
func deleteLocalClient(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	localClientID := r.FormValue("deleteLocalClient")

	id, err := strconv.Atoi(localClientID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	localClient, err := internal.GetClientByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteClient(db, localClient); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func listLocalClient(db *database.DB, r *http.Request) ([]*model.Client, FiltersPagination, string) {
	localClientFound := ""
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

	localClient, err := internal.ListClients(db, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, localClientFound
	}

	if search := urlParams.Get("search"); search != "" && searchLocalClient(search, localClient) == nil {
		localClientFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		localClientFound = "true"

		return []*model.Client{searchLocalClient(search, localClient)}, filter, localClientFound
	}

	filtersPtr, filterProtocol := protocolsFilter(r, &filter)
	paginationPage(&filter, len(localClient), r)

	if len(filterProtocol) > 0 {
		var localClients []*model.Client
		if localClients, err = internal.ListClients(db, "name", filter.OrderAsc, filter.Limit,
			filter.Offset*filter.Limit, filterProtocol...); err == nil {
			return localClients, *filtersPtr, localClientFound
		}
	}

	localClients, err := internal.ListClients(db, "name", filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, localClientFound
	}

	return localClients, filter, localClientFound
}

func searchLocalClient(localClientNameSearch string, listLocalClientSearch []*model.Client,
) *model.Client {
	for _, lc := range listLocalClientSearch {
		if lc.Name == localClientNameSearch {
			return lc
		}
	}

	return nil
}

//nolint:dupl // no similar func
func autocompletionLocalClientsFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")

		localClients, err := internal.GetClientsLike(db, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(localClients))
		for i, u := range localClients {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

//nolint:dupl // no similar func
func callMethodsLocalClientManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addLocalClientName") != "" {
		if newLocalClientErr := addLocalClient(db, r); newLocalClientErr != nil {
			logger.Errorf("failed to add localClient: %v", newLocalClientErr)

			return false, newLocalClientErr.Error(), "addLocalClientModal"
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteLocalClient") != "" {
		deleteLocalClientErr := deleteLocalClient(db, r)
		if deleteLocalClientErr != nil {
			logger.Errorf("failed to delete localClient: %v", deleteLocalClientErr)

			return false, deleteLocalClientErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editLocalClientID") != "" {
		idEdit := r.FormValue("editLocalClientID")

		id, err := strconv.Atoi(idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", ""
		}

		if editLocalClientErr := editLocalClient(db, r); editLocalClientErr != nil {
			logger.Errorf("failed to edit localClient: %v", editLocalClientErr)

			return false, editLocalClientErr.Error(), fmt.Sprintf("editLocalClientModal_%d", id)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func localClientManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := //nolint:forcetypeassert //u
			pageTranslated("local_client_management_page", userLanguage.(string)) //nolint:errcheck //u
		localClientList, filter, localClientFound := listLocalClient(db, r)

		value, errMsg, modalOpen := callMethodsLocalClientManagement(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := localClientManagementTemplate.ExecuteTemplate(w, "local_client_management_page", map[string]any{
			"myPermission":           myPermission,
			"tab":                    tabTranslated,
			"username":               user.Username,
			"language":               userLanguage,
			"localClient":            localClientList,
			"localClientFound":       localClientFound,
			"filter":                 filter,
			"currentPage":            currentPage,
			"TLSVersions":            TLSVersions,
			"CompatibilityModePeSIT": CompatibilityModePeSIT,
			"KeyExchanges":           sftp.ValidKeyExchanges,
			"Ciphers":                sftp.ValidCiphers,
			"MACs":                   sftp.ValidMACs,
			"errMsg":                 errMsg,
			"modalOpen":              modalOpen,
		}); tmplErr != nil {
			logger.Errorf("render local_client_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
