// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"io/fs"
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
	router.HandleFunc("/home", homePage(logger)).Methods("GET")
	router.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")

	subFS, err := fs.Sub(webFS, "front_end")
	if err != nil {
		logger.Error("error accessing css file: %v", err)

		return
	}

	router.PathPrefix("/static/").Handler(http.StripPrefix(Prefix+"/static/", http.FileServer(http.FS(subFS))))
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
