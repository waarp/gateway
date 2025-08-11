package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func statusServicesPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("status_services_page", userLanguage.(string)) //nolint:errcheck //u

		cores, servers, clients := internal.ListServices()

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
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
		}
		if r.URL.Query().Get("partial") == "true" {
			if err := statusServicesTemplate.ExecuteTemplate(w, "status_services_partial", data); err != nil {
				logger.Error("render status_services_partial: %v", err)
				http.Error(w, "Internal error", http.StatusInternalServerError)
			}

			return
		}

		if err := statusServicesTemplate.ExecuteTemplate(w, "status_services_page", data); err != nil {
			logger.Error("render status_services_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
