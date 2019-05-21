package admin

import (
	"net/http"

	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	"github.com/gorilla/mux"
)

// This is the access path to the status entry point
const statusURI string = "/status"

// Authentication checks if the request is authenticated using Basic HTTP
// authentication.
func Authentication(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Admin.Debugf("Received %s on %s", r.Method, r.URL)

			user, pswd, ok := r.BasicAuth()
			if !ok || user != "admin" || pswd != "adminpassword" {
				logger.Admin.Warningf("Authentication failed for user %s.", user)
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "Authentication failed", http.StatusUnauthorized)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// Function called when an HTTP request is received on the statusURI path.
// For now, it just send an OK status code.
func GetStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
