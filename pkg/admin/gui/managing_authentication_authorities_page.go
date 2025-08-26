package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func managingAuthenticationAuthoritiesPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("managing_authentication_authorities_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)

		if tmplErr := managingAuthenticationAuthoritiesTemplate.ExecuteTemplate(w, "managing_authentication_authorities_page",
			map[string]any{
				"myPermission":           myPermission,
				"tab":      tTranslated,
				"username": user.Username,
				"language": userLanguage,
			}); tmplErr != nil {
			logger.Errorf("render managing_authentication_authorities_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
