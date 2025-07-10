// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const Prefix = "/webui"

type ContextKey string

//nolint:gochecknoglobals // global
const (
	ContextUserKey     ContextKey = "user"
	ContextLanguageKey ContextKey = "language"
)

func GetUserByToken(r *http.Request, db database.ReadAccess) (*model.User, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return nil, fmt.Errorf("error cookie: %w", err)
	}

	value, ok := sessionStore.Load(cookie.Value)
	if !ok {
		return nil, errors.New("error loading session") //nolint:err113 // error
	}

	session, ok := value.(Session)
	if !ok {
		return nil, errors.New("internal error") //nolint:err113 // error
	}

	user, err := internal.GetUserByID(db, int64(session.UserID))
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	return user, nil
}

func LanguageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userLanguage := changeLanguage(w, r)
		ctx := context.WithValue(r.Context(), ContextLanguageKey, userLanguage)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AddGUIRouter(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)
	router.Use(LanguageMiddleware)
	// Add HTTP handlers to the router here.
	// Example:
	router.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")
	router.HandleFunc("/logout", logout()).Methods("GET")

	subFS, err := fs.Sub(webFS, "front-end")
	if err != nil {
		logger.Error("error accessing css file: %v", err)

		return
	}

	router.PathPrefix("/static/").Handler(http.StripPrefix(Prefix+"/static/", http.FileServer(http.FS(subFS))))

	secureRouter := router.PathPrefix("/").Subrouter()
	secureRouter.Use(AuthenticationMiddleware(logger, db))
	secureRouter.HandleFunc("/autocompletion/users", autocompletionFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/partners", autocompletionPartnersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/credentialPartner", autocompletionCredentialsPartnersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/remoteAccount", autocompletionRemoteAccountFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/credentialAccount", autocompletionCredentialsAccountsFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/servers", autocompletionServersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "home", http.StatusFound)
	})
	secureRouter.HandleFunc("/home", homePage(logger, db)).Methods("GET")
	secureRouter.HandleFunc("/user_management", userManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/partner_management", partnerManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/partner_authentication", partnerAuthenticationPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/remote_account_management", remoteAccountPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/account_authentication", accountAuthenticationPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/server_management", serverManagementPage(logger, db)).Methods("GET", "POST")
}

func logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secure := false

		token, err := r.Cookie("token")
		if err == nil {
			DeleteSession(token.Value)

			if r.TLS != nil {
				secure = true
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    "",
				Path:     "/",
				Expires:  time.Unix(0, 0),
				Secure:   secure,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		http.Redirect(w, r, "login", http.StatusFound)
	}
}

func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie("token")
			if err != nil || token.Value == "" {
				http.Redirect(w, r, "login", http.StatusFound)

				return
			}

			RefreshExpirationToken(token.Value, w, r)

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
