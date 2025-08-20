package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func cryptographicKeyManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("cryptographic_key_management_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		if tmplErr := cryptographicKeyManagementTemplate.ExecuteTemplate(w, "cryptographic_key_management_page",
			map[string]any{
				"tab":      tTranslated,
				"username": user.Username,
				"language": userLanguage,
			}); tmplErr != nil {
			logger.Errorf("render cryptographic_key_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
