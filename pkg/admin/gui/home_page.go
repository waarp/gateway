package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"
)

func homePage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("home_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert // userLanguage

		if err := templates.ExecuteTemplate(w, "home_page", map[string]any{
			"tab":      tabTranslated,
			"language": userLanguage,
		}); err != nil {
			logger.Error("render home_page: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
