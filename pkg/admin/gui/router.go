// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const Prefix = "/webui"

func AddGUIRouter(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)
	router.Use(AuthenticationMiddleware(logger, db))

	// Add HTTP handlers to the router here.
	// Example:
	router.HandleFunc("/home", homepage(db, logger)).Methods("GET")
}

func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check authentication here.

			// Called at the end if the authentication succeeded.
			next.ServeHTTP(w, r)
		})
	}
}
