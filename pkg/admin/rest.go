package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

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

func parseLimitOffsetOrder(r *http.Request, limit, offset *int, order *string,
	validSorting []string) error {

	sort.Strings(validSorting)
	if limStr := r.FormValue("limit"); limStr != "" {
		lim, err := strconv.Atoi(limStr)
		if err != nil {
			return &badRequest{msg: "'limit' must be an int"}
		}
		*limit = lim
	}
	if offStr := r.FormValue("offset"); offStr != "" {
		off, err := strconv.Atoi(offStr)
		if err != nil {
			return &badRequest{msg: "'offset' must be an int"}
		}
		*offset = off
	}
	if sortStr := r.FormValue("sortby"); sortStr != "" {
		if i := sort.SearchStrings(validSorting, sortStr); i != len(validSorting) {
			*order = sortStr
		} else {
			return &badRequest{msg: fmt.Sprintf("invalid value '%s' for parameter 'sortby'", sortStr)}
		}
	}
	if ordStr := r.FormValue("order"); ordStr != "" {
		if i := sort.SearchStrings(validOrders, ordStr); i != len(validOrders) {
			*order += " " + ordStr
		} else {
			return &badRequest{msg: fmt.Sprintf("invalid value '%s' for parameter 'order'", ordStr)}
		}
	} else {
		*order += " asc"
	}
	return nil
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

func readJSON(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return &badRequest{msg: err.Error()}
	}
	return nil
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

func restGet(acc database.Accessor, bean interface{}) error {
	if err := acc.Get(bean); err != nil {
		if err == database.ErrNotFound {
			return &notFound{}
		}
		return err
	}

	return nil
}

func restDelete(acc database.Accessor, bean interface{}) error {
	if exist, err := acc.Exists(bean); err != nil {
		return err
	} else if !exist {
		return &notFound{}
	}

	if err := acc.Delete(bean); err != nil {
		return err
	}

	return nil
}
