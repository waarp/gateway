package gui

import (
	"net/http"
	"sort"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func sortServicesByName(services []internal.Service) {
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
}

func statusServicesPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("status_services_page", userLanguage.(string)) //nolint:errcheck //u

		cores, servers, clients := internal.ListServices()

		sortServicesByName(cores)
		sortServicesByName(servers)
		sortServicesByName(clients)

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)

		data := map[string]any{
			"myPermission": myPermission,
			"tab":          tTranslated,
			"username":     user.Username,
			"language":     userLanguage,
			"cores":        cores,
			"servers":      servers,
			"clients":      clients,
			"Offline":      utils.StateOffline.String(),
			"Running":      utils.StateRunning.String(),
			"Error":        utils.StateError.String(),
		}
		if r.URL.Query().Get("partial") == True {
			if tmplErr := statusServicesTemplate.ExecuteTemplate(w, "status_services_partial", data); tmplErr != nil {
				logger.Errorf("render status_services_partial: %v", tmplErr)
				http.Error(w, "Internal error", http.StatusInternalServerError)
			}

			return
		}

		if tmplErr := statusServicesTemplate.ExecuteTemplate(w, "status_services_page", data); tmplErr != nil {
			logger.Errorf("render status_services_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
