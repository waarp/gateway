package rest

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

// AuthenticationMiddleware checks if the request is authenticated using Basic HTTP
// authentication.
func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debugf("Received %s on %q", r.Method, r.URL)

			login, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the request is missing credentials", http.StatusUnauthorized)

				return
			}

			var user model.User
			if err := db.Get(&user, "username=?", login).Owner().Run(); err != nil &&
				!database.IsNotFound(err) {
				logger.Errorf("Database error: %s", err)
				http.Error(w, "internal database error", http.StatusInternalServerError)

				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),
				[]byte(password)); err != nil {
				logger.Warningf("Invalid credentials for user %q", login)
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the given credentials are invalid", http.StatusUnauthorized)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter

	statusCode int
	errMsg     bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

//nolint:wrapcheck //wrapping the error adds nothing here
func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}

	if r.statusCode >= http.StatusBadRequest {
		if n, err := r.errMsg.Write(p); err != nil {
			return n, err
		}
	}

	return r.ResponseWriter.Write(p)
}

func LoggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _, _ := r.BasicAuth()
			logger.Debugf("Received %s request by user %q on %q", r.Method, user,
				r.URL.String())

			rec := responseRecorder{ResponseWriter: w}
			next.ServeHTTP(&rec, r)

			if rec.statusCode >= http.StatusBadRequest {
				logger.Warningf("Request failed with code %d: %s", rec.statusCode,
					rec.errMsg.String())
			} else {
				logger.Debugf("Responded with code %d", rec.statusCode)
			}
		})
	}
}

//nolint:gochecknoglobals //global var needed here for Waarp Transfer
var AppName = "waarp-gatewayd"

func ServerInfoMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", fmt.Sprintf("%s/%s", AppName, version.Num))
			w.Header().Set(api.DateHeader, time.Now().Format(time.RFC1123))

			next.ServeHTTP(w, r)
		})
	}
}
