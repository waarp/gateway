package admin

import (
	"encoding/json"
	"net/http"

	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// StatusURI is the access path to the status entry point
const StatusURI string = "/status"

// Authentication checks if the request is authenticated using Basic HTTP
// authentication.
func Authentication(logger *log.Logger, db *database.Db) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			invalidCredentials := func() {
				logger.Warning("Invalid credentials")
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			}
			internalError := func(err error) {
				msg := fmt.Sprintf("Internal error : %s", err)
				logger.Warning(msg)
				http.Error(w, msg, http.StatusInternalServerError)
			}

			logger.Debugf("Received %s on %s", r.Method, r.URL)

			login, pswd, ok := r.BasicAuth()
			if !ok {
				logger.Warning("Missing credentials")
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "Missing credentials", http.StatusUnauthorized)
				return
			}

			user := &database.User{
				Login: login,
			}
			if err := db.Get(user); err != nil {
				if err == database.ErrNotFound {
					invalidCredentials()
				} else {
					internalError(err)
				}
				return
			}

			if err := bcrypt.CompareHashAndPassword(user.Password, []byte(pswd)); err != nil {
				if err == bcrypt.ErrMismatchedHashAndPassword {
					invalidCredentials()
				} else {
					internalError(err)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Status is the status of the service
type Status struct {
	State  string
	Reason string
}

// Statuses maps a service name to its state
type Statuses map[string]Status

// GetStatus called when an HTTP request is received on the StatusURI path.
// For now, it just send an OK status code.
func GetStatus(services map[string]service.Servicer) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var statuses = make(Statuses)
		for name, serv := range services {
			code, reason := serv.State().Get()
			statuses[name] = Status{
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
