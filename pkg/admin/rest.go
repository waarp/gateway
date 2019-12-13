package admin

import (
	"encoding/json"
	"net/http"
	"sort"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/gorilla/mux"
)

const (
	// StatusPath is the access path to the status entry point.
	StatusPath = "/status"
	// LocalAgentsPath is the access path to the local servers entry point.
	LocalAgentsPath = "/servers"
	// RemoteAgentsPath is the access path to the partners entry point.
	RemoteAgentsPath = "/partners"
	// LocalAccountsPath is the access path to the local gateway accounts entry point.
	LocalAccountsPath = "/local_accounts"
	// RemoteAccountsPath is the access path to the distant partners accounts entry point.
	RemoteAccountsPath = "/remote_accounts"
	// CertificatesPath is the access path to the account certificates entry point.
	CertificatesPath = "/certificates"
	// TransfersPath is the access path to the transfers entry point.
	TransfersPath = "/transfers"
	// HistoryPath is the access path to the transfers history entry point.
	HistoryPath = "/history"
	// RulesPath is the access path to the transfers rules entry point.
	RulesPath = "/rules"
	// RulePermissionPath is the access path to the transfer rule permissions
	// entry point.
	RulePermissionPath = "/access"
	// RuleTasksPath is the access path to the transfer rule tasks entry point.
	RuleTasksPath = "/tasks"
)

var validOrders = []string{"asc", "desc"}

func init() {
	sort.Strings(validOrders)
}

type badRequest struct {
	msg string
}

func (e *badRequest) Error() string {
	return e.msg
}

type notFound struct{}

func (e *notFound) Error() string {
	return "Record not found"
}

func handleErrors(w http.ResponseWriter, logger *log.Logger, err error) {
	switch err.(type) {
	case *notFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	case *badRequest:
		http.Error(w, err.Error(), http.StatusBadRequest)
	case *database.ErrInvalid:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		logger.Warning(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, bean interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(bean)
}

// Authentication checks if the request is authenticated using Basic HTTP
// authentication.
func Authentication(logger *log.Logger, _ *database.Db) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = func() {
				logger.Warning("Invalid credentials")
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			}

			logger.Debugf("Received %s on %s", r.Method, r.URL)

			next.ServeHTTP(w, r)
		})
	}
}
