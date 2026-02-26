package gui

import (
	"net/http"
	"sort"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

func sortServicesByName(services []internal.Service) {
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
}

func statusServicesPage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		userLanguage := locale.GetLanguage(r)
		tTranslated := pageTranslated("status_services_page", userLanguage)
		myPermission := model.MaskToPerms(user.Permissions)

		cores, servers, clients := internal.ListServices()

		sortServicesByName(cores)
		sortServicesByName(servers)
		sortServicesByName(clients)

		data := map[string]any{
			"appName":      constants.AppName,
			"version":      version.Num,
			"compileDate":  version.Date,
			"revision":     version.Commit,
			"docLink":      constants.DocLink(userLanguage),
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
