package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

//nolint:dupl // no similar func (is for server)
func listCredentialServer(serverName string, db *database.DB, r *http.Request) (
	[]*model.Credential, Filters, string,
) {
	credentialServerFound := ""
	filter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	urlParams := r.URL.Query()

	filter.OrderAsc = urlParams.Get("orderAsc") == True

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

	serversCredentials, err := internal.ListServerCredentials(db, serverName, "name", true, 0, 0)
	if err != nil {
		return nil, Filters{}, credentialServerFound
	}

	if search := urlParams.Get("search"); search != "" && searchCredentialServer(search, serversCredentials) == nil {
		credentialServerFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialServerFound = True

		return []*model.Credential{searchCredentialServer(search, serversCredentials)}, filter, credentialServerFound
	}

	paginationPage(&filter, uint64(len(serversCredentials)), r)

	serversCredentialsList, err := internal.ListServerCredentials(db, serverName, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, credentialServerFound
	}

	return serversCredentialsList, filter, credentialServerFound
}

//nolint:dupl // is not the same, GetCredentialsLike is called
func autocompletionCredentialsServersFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		prefix := urlParams.Get("q")
		var err error
		var server *model.LocalAgent
		var idA uint64

		serverID := urlParams.Get("serverID")

		if serverID != "" {
			idA, err = strconv.ParseUint(serverID, 10, 64)
			if err != nil {
				http.Error(w, "failed to convert server id to int", http.StatusInternalServerError)

				return
			}

			server, err = internal.GetServerByID(db, int64(idA))
			if err != nil {
				http.Error(w, "failed to get server by id", http.StatusInternalServerError)

				return
			}
		}

		credentialsServers, err := internal.GetCredentialsLike(db, server, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(credentialsServers))
		for i, u := range credentialsServers {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchCredentialServer(credentialServerNameSearch string,
	listCredentialServerSearch []*model.Credential,
) *model.Credential {
	for _, cs := range listCredentialServerSearch {
		if cs.Name == credentialServerNameSearch {
			return cs
		}
	}

	return nil
}

func addCredentialServer(serverName string, db *database.DB, r *http.Request) error {
	var newCredentialServer model.Credential

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newCredentialServerName := r.FormValue("addCredentialServerName"); newCredentialServerName != "" {
		newCredentialServer.Name = newCredentialServerName
	}

	if newCredentialServerType := r.FormValue("addCredentialServerType"); newCredentialServerType != "" {
		newCredentialServer.Type = newCredentialServerType
	}

	switch newCredentialServer.Type {
	case auth.Password:
		newCredentialServer.Value = r.FormValue("addCredentialValue")
	case sftp.AuthSSHPrivateKey:
		newCredentialServer.Value = r.FormValue("addCredentialValueFile")
	case auth.TLSCertificate:
		newCredentialServer.Value = r.FormValue("addCredentialValueFile1")
		newCredentialServer.Value2 = r.FormValue("addCredentialValueFile2")
	case pesit.PreConnectionAuth:
		newCredentialServer.Value = r.FormValue("addCredentialValue1")
		newCredentialServer.Value2 = r.FormValue("addCredentialValue2")
	}

	server, err := internal.GetServer(db, serverName)
	if err != nil {
		return fmt.Errorf("failed to get server server: %w", err)
	}

	server.SetCredOwner(&newCredentialServer)

	if addErr := internal.InsertCredential(db, &newCredentialServer); addErr != nil {
		return fmt.Errorf("failed to add credential server: %w", addErr)
	}

	return nil
}

//nolint:dupl // no similar func (is for server)
func editCredentialServer(serverName string, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialServerID := r.FormValue("editCredentialServerID")

	id, err := strconv.ParseUint(credentialServerID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editCredentialServer, err := internal.GetServerCredentialByID(db, serverName, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get credential server: %w", err)
	}

	if editCredentialServerName := r.FormValue("editCredentialServerName"); editCredentialServerName != "" {
		editCredentialServer.Name = editCredentialServerName
	}

	if editCredentialServerType := r.FormValue("editCredentialServerType"); editCredentialServerType != "" {
		editCredentialServer.Type = editCredentialServerType
	}

	switch editCredentialServer.Type {
	case auth.Password:
		editCredentialServer.Value = r.FormValue("editCredentialValue")
	case sftp.AuthSSHPrivateKey:
		editCredentialServer.Value = r.FormValue("editCredentialValueFile")
	case auth.TLSCertificate:
		editCredentialServer.Value = r.FormValue("editCredentialValueFile1")
		editCredentialServer.Value2 = r.FormValue("editCredentialValueFile2")
	case pesit.PreConnectionAuth:
		editCredentialServer.Value = r.FormValue("editCredentialValue1")
		editCredentialServer.Value2 = r.FormValue("editCredentialValue2")
	}

	if err = internal.UpdateCredential(db, editCredentialServer); err != nil {
		return fmt.Errorf("failed to edit credential server: %w", err)
	}

	return nil
}

//nolint:dupl // no similar func (is for server)
func deleteCredentialServer(serverName string, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialServerID := r.FormValue("deleteCredentialServer")

	id, err := strconv.ParseUint(credentialServerID, 10, 64)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	credentialServer, err := internal.GetServerCredentialByID(db, serverName, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeleteCredential(db, credentialServer); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

//nolint:dupl // method for server authentication
func callMethodsServerAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	server *model.LocalAgent,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("deleteCredentialServer") != "" {
		deleteCredentialServerErr := deleteCredentialServer(server.Name, db, r)
		if deleteCredentialServerErr != nil {
			logger.Errorf("failed to delete credential server: %v", deleteCredentialServerErr)

			return false, deleteCredentialServerErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?&serverID=%d", r.URL.Path, server.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("addCredentialServerName") != "" {
		addCredentialServerErr := addCredentialServer(server.Name, db, r)
		if addCredentialServerErr != nil {
			logger.Errorf("failed to add credential server: %v", addCredentialServerErr)

			return false, addCredentialServerErr.Error(), "addCredentialServerModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d", r.URL.Path, server.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editCredentialServerID") != "" {
		idEdit := r.FormValue("editCredentialServerID")

		id, err := strconv.ParseUint(idEdit, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", ""
		}

		editCredentialServerErr := editCredentialServer(server.Name, db, r)
		if editCredentialServerErr != nil {
			logger.Errorf("failed to edit credential server: %v", editCredentialServerErr)

			return false, editCredentialServerErr.Error(), fmt.Sprintf("editCredentialExternalModal_%d", id)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d", r.URL.Path, server.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func serverAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("server_authentication_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var server *model.LocalAgent
		var id uint64

		serverID := r.URL.Query().Get("serverID")
		if serverID != "" {
			id, err = strconv.ParseUint(serverID, 10, 64)
			if err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			server, err = internal.GetServerByID(db, int64(id))
			if err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		serversCredentials, filter, credentialServerFound := listCredentialServer(server.Name, db, r)

		value, errMsg, modalOpen := callMethodsServerAuthentication(logger, db, w, r, server)
		if value {
			return
		}

		listSupportedProtocol := supportedProtocolExternal(server.Protocol)
		listSupportedProtocol = slices.DeleteFunc(listSupportedProtocol, func(method_auth string) bool {
			return method_auth == pesit.PreConnectionAuth
		})
		currentPage := filter.Offset + 1

		if tmplErr := serverAuthenticationTemplate.ExecuteTemplate(w, "server_authentication_page", map[string]any{
			"myPermission":          myPermission,
			"tab":                   tTranslated,
			"username":              user.Username,
			"language":              userLanguage,
			"server":                server,
			"serverCredentials":     serversCredentials,
			"listSupportedProtocol": listSupportedProtocol,
			"filter":                filter,
			"currentPage":           currentPage,
			"credentialServerFound": credentialServerFound,
			"errMsg":                errMsg,
			"modalOpen":             modalOpen,
			"hasServerID":           true,
		}); tmplErr != nil {
			logger.Errorf("render server_authentication_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
