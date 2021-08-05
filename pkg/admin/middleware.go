package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// authentication checks if the request is authenticated using Basic HTTP
// authentication.
func authentication(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debugf("Received %s on %s", r.Method, r.URL)

			login, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the request is missing credentials", http.StatusUnauthorized)
				return
			}

			var user model.User
			if err := db.Get(&user, "username=? AND owner=?", login, conf.GlobalConfig.ServerConf.GatewayName).
				Run(); err != nil {
				if database.IsNotFound(err) {
					logger.Warningf("Invalid authentication for user '%s'", login)
					w.Header().Set("WWW-Authenticate", "Basic")
					http.Error(w, "the given credentials are invalid", http.StatusUnauthorized)
					return
				}
				logger.Criticalf("Database error: %s", err)
				http.Error(w, "internal database error", http.StatusInternalServerError)
				return
			}

			if err := bcrypt.CompareHashAndPassword(user.Password, []byte(password)); err != nil {
				logger.Warningf("Invalid password for user '%s'", login)
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the given credentials are invalid", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
