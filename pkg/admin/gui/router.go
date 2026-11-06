// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/login"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

const Prefix = constants.WebuiPrefix

//nolint:gochecknoglobals // global
const (
	ContextUserKey     = constants.ContextUserKey
	ContextLanguageKey = constants.ContextLanguageKey
)

func LanguageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userLanguage := changeLanguage(w, r)
		ctx := context.WithValue(r.Context(), ContextLanguageKey, userLanguage)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RouterPages registers the screens of the web interface and the former
// addresses of its historic pages.
func RouterPages(secureRouter *mux.Router, db *database.DB, logger *log.Logger) {
	routeFormerPages(secureRouter, db)
	backend.MakeRouter(secureRouter, db, logger)
}

func AddGUIRouter(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)
	router.Use(LanguageMiddleware)
	// Add HTTP handlers to the router here.
	// Example:
	router.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")
	// The kit chrome signs out with a POST form; the GET route, which the
	// historic pages linked, stays for the addresses saved since.
	router.HandleFunc("/logout", logout()).Methods("GET", "POST")

	backend.MakeKitStaticRouter(router)

	secureRouter := router.PathPrefix("/").Subrouter()
	secureRouter.Use(AuthenticationMiddleware(logger, db))
	secureRouter.StrictSlash(true)

	secureRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, pagePath(homePath), http.StatusFound)
	})

	RouterPages(secureRouter, db, logger)
}

// logout ends the session and leads to the sign-in page, which says so.
func logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(tokenCookieName); err == nil {
			invalidateToken(cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     tokenCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		signIn := url.URL{
			Path:     pagePath(loginPath),
			RawQuery: url.Values{login.ReasonParam: {login.ReasonSignedOut}}.Encode(),
		}
		http.Redirect(w, r, signIn.String(), http.StatusSeeOther)
	}
}

// AuthenticationMiddleware lets a request through with its user when its
// session cookie is valid, and leads to the sign-in page otherwise, with the
// page asked for; a cookie that was given and refused (expired, replaced by a
// sign-in elsewhere, lost at a restart) is said to the user by the reason.
func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			params := url.Values{}
			if curPage := r.URL.RequestURI(); curPage != Prefix+"/" {
				params.Set(redirectParam, curPage)
			}

			toSignIn := func(reason string) {
				if reason != "" {
					params.Set(login.ReasonParam, reason)
				}

				http.Redirect(w, r, (&url.URL{Path: pagePath(loginPath), RawQuery: params.Encode()}).String(), http.StatusFound)
			}

			tokenCookie, err := r.Cookie(tokenCookieName)
			if err != nil || tokenCookie.Value == "" {
				toSignIn("")

				return
			}

			user, err := validateSession(db, tokenCookie.Value)
			if err != nil {
				logger.Infof("Failed to validate session: %v", err)
				toSignIn(login.ReasonExpired)

				return
			}

			// A refresh made by a page on its own (historic "partial", v2
			// "fragment") is not a user action: it does not extend the session.
			if query := r.URL.Query(); query.Get("partial") == "" && query.Get(constants.FragmentParam) == "" {
				refreshToken(r, w, tokenCookie.Value)
			}

			ctx := context.WithValue(r.Context(), ContextUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
