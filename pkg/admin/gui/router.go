// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"context"
	"io/fs"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"
)

const Prefix = "/webui"

type ContextKey string

const ContextUserKey ContextKey = "user"

func AddGUIRouter(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)

	// Add HTTP handlers to the router here.
	// Example:
	router.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")

	subFS, err := fs.Sub(webFS, "front_end")
	if err != nil {
		logger.Error("error accessing css file: %v", err)

		return
	}

	router.PathPrefix("/static/").Handler(http.StripPrefix(Prefix+"/static/", http.FileServer(http.FS(subFS))))

	secureRouter := router.PathPrefix("/").Subrouter()
	secureRouter.Use(AuthenticationMiddleware(logger, db))
	secureRouter.HandleFunc("/home", homePage(logger)).Methods("GET")
}

func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie("token")
			if err != nil || token.Value == "" {
				http.Redirect(w, r, "login", http.StatusFound)

				return
			}

			userID, found := ValidateSession(token.Value)
			if !found {
				http.Redirect(w, r, "login", http.StatusFound)

				return
			}

			user, err := internal.GetUserByID(db, int64(userID))
			if err != nil {
				logger.Error("erreur: %v", err)
				http.Error(w, "erreur interne", http.StatusInternalServerError)

				return
			}

			ctx := context.WithValue(r.Context(), ContextUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
