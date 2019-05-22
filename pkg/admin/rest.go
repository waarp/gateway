package admin

import (
	"encoding/json"
	"net/http"

	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	"code.waarp.fr/waarp/gateway-ng/pkg/tk/service"
	"github.com/gorilla/mux"
)

// This is the access path to the status entry point
const StatusURI string = "/status"

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

type status struct {
	State  string
	Reason string
}

// Function called when an HTTP request is received on the StatusURI path.
// For now, it just send an OK status code.
func GetStatus(services map[string]service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var statuses = make(map[string]status)
		for name, serv := range services {
			code, reason := serv.State().Get()
			statuses[name] = status{
				State:  code.Name(),
				Reason: reason,
			}
		}
		resp, err := json.Marshal(statuses)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
