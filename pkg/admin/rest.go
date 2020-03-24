package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// Authentication checks if the request is authenticated using Basic HTTP
// authentication.
func Authentication(logger *log.Logger, db *database.Db) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = func() {
				logger.Warning("Invalid credentials")
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			}
			logger.Debugf("Received %s on %s", r.Method, r.URL)

			login, password, ok := r.BasicAuth()
			if !ok {
				http.Error(w, "Missing credentials", http.StatusUnauthorized)
				return
			}

			user := &model.User{Username: login, Owner: database.Owner}
			if err := db.Get(user); err != nil {
				if err == database.ErrNotFound {
					logger.Warningf("Invalid authentication for user '%s'", login)
					http.Error(w, "Invalid credentials", http.StatusUnauthorized)
					return
				}
				logger.Criticalf("Database error: %s", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			if err := bcrypt.CompareHashAndPassword(user.Password, []byte(password)); err != nil {
				logger.Warningf("Invalid password for user '%s'", login)
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
