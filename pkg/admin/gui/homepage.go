package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"
)

func homepage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "home_page", map[string]any{"Title": "Accueil"}); err != nil {
			logger.Error("render homepage: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
