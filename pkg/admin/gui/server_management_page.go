package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func listServer(db *database.DB, r *http.Request) ([]*model.LocalAgent, FiltersPagination, string) {
	serverFound := ""
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

	servers, err := internal.ListServers(db, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, serverFound
	}

	if search := urlParams.Get("search"); search != "" && searchServer(search, servers) == nil {
		serverFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		serverFound = "true"

		return []*model.LocalAgent{searchServer(search, servers)}, filter, serverFound
	}

	paginationPage(&filter, len(servers), r)

	serversList, err := internal.ListServers(db, "name",
		filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, serverFound
	}

	return serversList, filter, serverFound
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

func serverManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("server_management_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u
		serverList, filter, serverFound := listServer(db, r)

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if err := serverManagementTemplate.ExecuteTemplate(w, "server_management_page", map[string]any{
			"myPermission": myPermission,
			"tab":          tabTranslated,
			"username":     user.Username,
			"language":     userLanguage,
			"server":       serverList,
			"serverFound":  serverFound,
			"filter":       filter,
			"currentPage":  currentPage,
		}); err != nil {
			logger.Error("render server_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
