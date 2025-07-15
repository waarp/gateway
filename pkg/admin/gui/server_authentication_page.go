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

//nolint:dupl // no similar func (is for server)
func listCredentialServer(serverName string, db *database.DB, r *http.Request) (
	[]*model.Credential, FiltersPagination, string,
) {
	credentialServerFound := ""
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

	serversCredentials, err := internal.ListServerCredentials(db, serverName, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, credentialServerFound
	}

	if search := urlParams.Get("search"); search != "" && searchCredentialServer(search, serversCredentials) == nil {
		credentialServerFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		credentialServerFound = "true"

		return []*model.Credential{searchCredentialServer(search, serversCredentials)}, filter, credentialServerFound
	}

	paginationPage(&filter, len(serversCredentials), r)

	serversCredentialsList, err := internal.ListServerCredentials(db, serverName, "name",
		filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, credentialServerFound
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
		var idA int

		serverID := urlParams.Get("serverID")

		if serverID != "" {
			idA, err = strconv.Atoi(serverID)
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

		if err := json.NewEncoder(w).Encode(names); err != nil {
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
	case "password":
		newCredentialServer.Value = r.FormValue("addCredentialValue")
	case "ssh_private_key":
		newCredentialServer.Value = r.FormValue("addCredentialValueFile")
	case "tls_certificate":
		newCredentialServer.Value = r.FormValue("addCredentialValueFile1")
		newCredentialServer.Value2 = r.FormValue("addCredentialValueFile2")
	case "pesit_pre-connection_auth":
		newCredentialServer.Value = r.FormValue("addCredentialValue1")
		newCredentialServer.Value2 = r.FormValue("addCredentialValue2")
	}

	server, err := internal.GetServer(db, serverName)
	if err != nil {
		return fmt.Errorf("failed to get server server: %w", err)
	}

	server.SetCredOwner(&newCredentialServer)

	if err := internal.InsertCredential(db, &newCredentialServer); err != nil {
		return fmt.Errorf("failed to add credential server: %w", err)
	}

	return nil
}

//nolint:dupl // no similar func (is for server)
func editCredentialServer(serverName string, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	credentialServerID := r.FormValue("editCredentialServerID")

	id, err := strconv.Atoi(credentialServerID)
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
	case "password":
		editCredentialServer.Value = r.FormValue("editCredentialValue")
	case "ssh_private_key":
		editCredentialServer.Value = r.FormValue("editCredentialValueFile")
	case "tls_certificate":
		editCredentialServer.Value = r.FormValue("editCredentialValueFile1")
		editCredentialServer.Value2 = r.FormValue("editCredentialValueFile2")
	case "pesit_pre-connection_auth":
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

	id, err := strconv.Atoi(credentialServerID)
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

func callMethodsServerAuthentication(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	server *model.LocalAgent,
) (bool, string, string) {
	if r.Method == http.MethodPost && r.FormValue("deleteCredentialServer") != "" {
		deleteCredentialServerErr := deleteCredentialServer(server.Name, db, r)
		if deleteCredentialServerErr != nil {
			logger.Error("failed to delete credential server: %v", deleteCredentialServerErr)

			return false, deleteCredentialServerErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?&serverID=%d", r.URL.Path, server.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("addCredentialServerName") != "" {
		addCredentialServerErr := addCredentialServer(server.Name, db, r)
		if addCredentialServerErr != nil {
			logger.Error("failed to add credential server: %v", addCredentialServerErr)

			return false, addCredentialServerErr.Error(), "addCredentialServerModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?serverID=%d", r.URL.Path, server.ID),
			http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editCredentialServerID") != "" {
		idEdit := r.FormValue("editCredentialServerID")

		id, err := strconv.Atoi(idEdit)
		if err != nil {
			logger.Error("failed to convert id to int: %v", err)

			return false, "", ""
		}

		editCredentialServerErr := editCredentialServer(server.Name, db, r)
		if editCredentialServerErr != nil {
			logger.Error("failed to edit credential server: %v", editCredentialServerErr)

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
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var server *model.LocalAgent
		var id int

		serverID := r.URL.Query().Get("serverID")
		if serverID != "" {
			id, err = strconv.Atoi(serverID)
			if err != nil {
				logger.Error("failed to convert id to int: %v", err)
			}

			server, err = internal.GetServerByID(db, int64(id))
			if err != nil {
				logger.Error("failed to get id: %v", err)
			}
		}

		serversCredentials, filter, credentialServerFound := listCredentialServer(server.Name, db, r)

		value, errMsg, modalOpen := callMethodsServerAuthentication(logger, db, w, r, server)
		if value {
			return
		}

		listSupportedProtocol := supportedProtocolExternal(server.Protocol)
		currentPage := filter.Offset + 1

		if err := serverAuthenticationTemplate.ExecuteTemplate(w, "server_authentication_page", map[string]any{
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
		}); err != nil {
			logger.Error("render server_authentication_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
