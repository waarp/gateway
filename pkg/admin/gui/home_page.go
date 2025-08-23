//nolint:dupl // in progress
package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func homePage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("home_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert // userLanguage

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error loading user session: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)

		if tmplErr := homeTemplate.ExecuteTemplate(w, "home_page", map[string]any{
			"myPermission": myPermission,
			"tab":          tabTranslated,
			"username":     user.Username,
			"language":     userLanguage,
		}); tmplErr != nil {
			logger.Errorf("render home_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
