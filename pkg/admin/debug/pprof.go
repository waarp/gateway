// Package debug provides the HTTP handlers for debugging the gateway.
package debug

import (
	"net/http"
	"net/http/pprof"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const Prefix = "/debug"

func AddDebugHandler(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)
	router.Use(
		AuthenticationMiddleware(logger, db),
	)

	router.HandleFunc("/pprof/", pprof.Index)
	router.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/pprof/profile", pprof.Profile)
	router.HandleFunc("/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/pprof/trace", pprof.Trace)

	router.HandleFunc("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	router.HandleFunc("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	router.HandleFunc("/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
	router.HandleFunc("/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	router.HandleFunc("/pprof/block", pprof.Handler("block").ServeHTTP)
	router.HandleFunc("/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
}

func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("Received %s on %s", r.Method, r.URL)

			login, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the request is missing credentials", http.StatusUnauthorized)

				return
			}

			var user model.User
			if err := db.Get(&user, "username=?", login).Owner().Run(); err != nil &&
				!database.IsNotFound(err) {
				logger.Error("Database error: %v", err)
				http.Error(w, "internal database error", http.StatusInternalServerError)

				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),
				[]byte(password)); err != nil {
				logger.Warning("Invalid credentials for user %q", login)
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the given credentials are invalid", http.StatusUnauthorized)

				return
			}

			if !user.Permissions.HasPermission(model.PermAdminRead) {
				logger.Warning("Unauthorized access for user %q", login)
				http.Error(w, "the given credentials are invalid", http.StatusForbidden)
			}

			next.ServeHTTP(w, r)
		})
	}
}
