package gui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:dupl,cyclop // is not similar, is method for local client
func addLocalClient(db *database.DB, r *http.Request) error {
	var newLocalClient model.Client

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newLocalClientName := r.FormValue("addLocalClientName"); newLocalClientName != "" {
		newLocalClient.Name = newLocalClientName
	}

	newLocalClient.Disabled = r.FormValue("addNoAutoStart") == True

	if newLocalClientProtocol := r.FormValue("addLocalClientProtocol"); newLocalClientProtocol != "" {
		newLocalClient.Protocol = newLocalClientProtocol
	}

	if newLocalClientHost := r.FormValue("addLocalClientHost"); newLocalClientHost != "" {
		newLocalClient.LocalAddress.Host = newLocalClientHost
	}

	if newLocalClientPort := r.FormValue("addLocalClientPort"); newLocalClientPort != "" {
		port, err := internal.ParseUint[uint16](newLocalClientPort)
		if err != nil {
			return fmt.Errorf("failed to get port: %w", err)
		}
		newLocalClient.LocalAddress.Port = port
	}

	if nbOfAttempts := r.FormValue("nbOfAttempts"); nbOfAttempts != "" {
		remainingTries, err := internal.ParseInt[int8](nbOfAttempts)
		if err != nil {
			return fmt.Errorf("failed to parse remainingTries in int: %w", err)
		}
		newLocalClient.NbOfAttempts = remainingTries
	}

	retryDelay := ""
	if h := r.FormValue("retryDelaypH"); h != "" {
		retryDelay += h + "h"
	}

	if m := r.FormValue("retryDelayM"); m != "" {
		retryDelay += m + "m"
	}

	if s := r.FormValue("retryDelayS"); s != "" {
		retryDelay += s + "s"
	}

	if retryDelay != "" {
		firstRetryDelay, err := time.ParseDuration(retryDelay)
		if err == nil {
			newLocalClient.FirstRetryDelay = int32(firstRetryDelay.Seconds())
		}
	}

	if retryIncrementFactor := r.FormValue("retryIncrementFactor"); retryIncrementFactor != "" {
		retryIncrementFloat, err := strconv.ParseFloat(retryIncrementFactor, 32)
		if err != nil {
			return fmt.Errorf("failed to parse retryIncrement in float: %w", err)
		}
		newLocalClient.RetryIncrementFactor = float32(retryIncrementFloat)
	}

	switch newLocalClient.Protocol {
	case r66.R66, r66.R66TLS:
		newLocalClient.ProtoConfig = protoConfigR66Client(r, newLocalClient.Protocol)
	case httpconst.HTTP, httpconst.HTTPS:
		newLocalClient.ProtoConfig = protoConfigHTTPclient(r, newLocalClient.Protocol)
	case sftp.SFTP:
		newLocalClient.ProtoConfig = protoConfigSFTPClient(r)
	case ftp.FTP, ftp.FTPS:
		newLocalClient.ProtoConfig = protoConfigFTPClient(r, newLocalClient.Protocol)
	case pesit.Pesit, pesit.PesitTLS:
		newLocalClient.ProtoConfig = protoConfigPeSITClient(r)
	case webdav.Webdav:
		newLocalClient.ProtoConfig = protoConfigWebdavClient(r)
	case webdav.WebdavTLS:
		newLocalClient.ProtoConfig = protoConfigWebdavTLSClient(r)
	}

	if err := internal.AddClient(db, &newLocalClient); err != nil {
		return fmt.Errorf("failed to add local client: %w", err)
	}

	return nil
}

//nolint:dupl,cyclop // is not similar, is method for local client
func editLocalClient(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	localClientID := r.FormValue("editLocalClientID")

	id, err := internal.ParseUint[uint64](localClientID)
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

	editLocalClient.Disabled = r.FormValue("editNoAutoStart") == True

	if editLocalClientProtocol := r.FormValue("editLocalClientProtocol"); editLocalClientProtocol != "" {
		editLocalClient.Protocol = editLocalClientProtocol
	}

	editLocalClient.LocalAddress.Host = r.FormValue("editLocalClientHost")

	if editLocalClientPort := r.FormValue("editLocalClientPort"); editLocalClientPort != "" {
		port, portErr := internal.ParseUint[uint16](editLocalClientPort)
		if portErr != nil {
			return fmt.Errorf("failed to get port: %w", portErr)
		}
		editLocalClient.LocalAddress.Port = port
	}

	editNbOfAttempts := r.FormValue("editNbOfAttempts")

	attempts, attemptsErr := internal.ParseInt[int8](editNbOfAttempts)
	if attemptsErr != nil {
		return fmt.Errorf("failed to parse remainingTries in int: %w", attemptsErr)
	}
	editLocalClient.NbOfAttempts = attempts

	editRetryDelay := ""
	if h := r.FormValue("editRetryDelayH"); h != "" {
		editRetryDelay += h + "h"
	}

	if m := r.FormValue("editRetryDelayM"); m != "" {
		editRetryDelay += m + "m"
	}

	if s := r.FormValue("editRetryDelayS"); s != "" {
		editRetryDelay += s + "s"
	}

	firstRetryDelay, err := time.ParseDuration(editRetryDelay)
	if err == nil && editRetryDelay != "" {
		editLocalClient.FirstRetryDelay = int32(firstRetryDelay.Seconds())
	} else {
		editLocalClient.FirstRetryDelay = 0
	}

	editRetryIncrementFactor := r.FormValue("editRetryIncrementFactor")

	retryIncrementFloat, retryErr := strconv.ParseFloat(editRetryIncrementFactor, 32)
	if retryErr != nil {
		return fmt.Errorf("failed to parse retryIncrement in float: %w", retryErr)
	}
	editLocalClient.RetryIncrementFactor = float32(retryIncrementFloat)

	switch editLocalClient.Protocol {
	case r66.R66, r66.R66TLS:
		editLocalClient.ProtoConfig = protoConfigR66Client(r, editLocalClient.Protocol)
	case httpconst.HTTP, httpconst.HTTPS:
		editLocalClient.ProtoConfig = protoConfigHTTPclient(r, editLocalClient.Protocol)
	case sftp.SFTP:
		editLocalClient.ProtoConfig = protoConfigSFTPClient(r)
	case ftp.FTP, ftp.FTPS:
		editLocalClient.ProtoConfig = protoConfigFTPClient(r, editLocalClient.Protocol)
	case pesit.Pesit, pesit.PesitTLS:
		editLocalClient.ProtoConfig = protoConfigPeSITClient(r)
	case webdav.Webdav:
		editLocalClient.ProtoConfig = protoConfigWebdavClient(r)
	case webdav.WebdavTLS:
		editLocalClient.ProtoConfig = protoConfigWebdavTLSClient(r)
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

	id, err := internal.ParseUint[uint64](localClientID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	localClient, err := internal.GetClientByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.RemoveClient(r.Context(), db, localClient); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func listLocalClient(db *database.DB, r *http.Request) ([]*model.Client, *Filters, string) {
	localClientFound := ""
	defaultFilter := &Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "local_client_management_page"); ok {
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
		if l, err := internal.ParseUint[uint64](limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := internal.ParseUint[uint64](offsetRes); err == nil {
			filter.Offset = o
		}
	}

	localClient, err := internal.ListClients(db, "name", true, 0, 0)
	if err != nil {
		return nil, nil, localClientFound
	}

	if search := urlParams.Get("search"); search != "" && searchLocalClient(search, localClient) == nil {
		localClientFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		localClientFound = True

		return []*model.Client{searchLocalClient(search, localClient)}, filter, localClientFound
	}
	filtersPtr, filterProtocol := checkProtocolsFilter(r, isApply, filter)
	paginationPage(filter, uint64(len(localClient)), r)

	if len(filterProtocol) > 0 {
		var localClients []*model.Client
		if localClients, err = internal.ListClients(db, "name", filter.OrderAsc, int(filter.Limit),
			int(filter.Offset*filter.Limit), filterProtocol...); err == nil {
			return localClients, filtersPtr, localClientFound
		}
	}

	localClients, err := internal.ListClients(db, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, nil, localClientFound
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
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("addLocalClientName") != "" {
		if newLocalClientErr := addLocalClient(db, r); newLocalClientErr != nil {
			logger.Errorf("failed to add localClient: %v", newLocalClientErr)
			modalElement = getFormValues(r)

			return false, newLocalClientErr.Error(), "addLocalClientModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("deleteLocalClient") != "" {
		deleteLocalClientErr := deleteLocalClient(db, r)
		if deleteLocalClientErr != nil {
			logger.Errorf("failed to delete localClient: %v", deleteLocalClientErr)

			return false, deleteLocalClientErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editLocalClientID") != "" {
		idEdit := r.FormValue("editLocalClientID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		if editLocalClientErr := editLocalClient(db, r); editLocalClientErr != nil {
			logger.Errorf("failed to edit localClient: %v", editLocalClientErr)
			modalElement = getFormValues(r)

			return false, editLocalClientErr.Error(), fmt.Sprintf("editLocalClientModal_%d", id), modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("switchClientStatus") != "" {
		statusClientErr := switchClientStatus(db, r)
		if statusClientErr != nil {
			logger.Errorf("failed to switch client status: %v", statusClientErr)

			return false, statusClientErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func switchClientStatus(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	clientID := r.FormValue("switchClientStatus")

	id, err := internal.ParseUint[uint64](clientID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	client, err := internal.GetClientByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get client by id: %w", err)
	}

	state, _ := internal.GetClientStatus(client)

	if state == utils.StateOffline || state == utils.StateError {
		if restartErr := internal.RestartClient(r.Context(), db, client); restartErr != nil {
			return fmt.Errorf("failed to restart client: %w", restartErr)
		}
	}

	if state == utils.StateRunning {
		if stopErr := internal.StopClient(r.Context(), client); stopErr != nil {
			return fmt.Errorf("failed to stop client: %w", stopErr)
		}
	}

	return nil
}

func localClientManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		userLanguage := locale.GetLanguage(r)
		tabTranslated := pageTranslated("local_client_management_page", userLanguage)
		localClientList, filter, localClientFound := listLocalClient(db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "local_client_management_page", filter)

		value, errMsg, modalOpen, modalElement := callMethodsLocalClientManagement(logger, db, w, r)
		if value {
			return
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		localClientManagementTemplate := template.Must(
			template.New("local_client_management_page.html").
				Funcs(CombinedFuncMap(db)).
				ParseFS(webFS, index, header, sidebar, addProtoConfig, editProtoConfig, displayProtoConfig,
					"front-end/html/local_client_management_page.html"),
		)

		if tmplErr := localClientManagementTemplate.ExecuteTemplate(w, "local_client_management_page", map[string]any{
			"appName":                constants.AppName,
			"version":                version.Num,
			"compileDate":            version.Date,
			"revision":               version.Commit,
			"docLink":                constants.DocLink(userLanguage),
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
			"protocolsList":          ProtocolsList(),
			"errMsg":                 errMsg,
			"modalOpen":              modalOpen,
			"modalElement":           modalElement,
		}); tmplErr != nil {
			logger.Errorf("render local_client_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
