package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"
)

func homePage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)

		if err := templates.ExecuteTemplate(w, "home_page", map[string]any{
			"Title":    "Accueil",
			"language": userLanguage,
		}); err != nil {
			logger.Error("render home_page: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
