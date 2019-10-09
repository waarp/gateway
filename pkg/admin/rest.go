package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
	"github.com/gorilla/mux"
)

const (
	// StatusPath is the access path to the status entry point
	StatusPath = "/status"
	// LocalAgentsPath is the access path to the local servers entry point
	LocalAgentsPath = "/servers"
	// RemoteAgentsPath is the access path to the partners entry point
	RemoteAgentsPath = "/partners"
	// LocalAccountsPath is the access path to the local gateway accounts entry point
	LocalAccountsPath = "/local_accounts"
	// RemoteAccountsPath is the access path to the distant partners accounts entry point
	RemoteAccountsPath = "/remote_accounts"
	// CertificatesPath is the access path to the account certificates entry point
	CertificatesPath = "/certificates"
	// TransfersPath is the access path to the transfers entry point
	TransfersPath = "/transfers"
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

func parseID(r *http.Request, param string) (uint64, error) {
	id, err := strconv.ParseUint(mux.Vars(r)[param], 10, 64)
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, &notFound{}
	}
	return id, nil
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

func restGet(db *database.Db, bean interface{}) error {
	if err := db.Get(bean); err != nil {
		if err == database.ErrNotFound {
			return &notFound{}
		}
		return err
	}

	return nil
}

func restCreate(db *database.Db, r *http.Request, bean interface{}) error {
	if err := readJSON(r, bean); err != nil {
		return err
	}

	if err := db.Create(bean); err != nil {
		return err
	}
	return nil
}

func restDelete(db *database.Db, bean interface{}) error {
	if exist, err := db.Exists(bean); err != nil {
		return err
	} else if !exist {
		return &notFound{}
	}

	if err := db.Delete(bean); err != nil {
		return err
	}

	return nil
}

func restUpdate(db *database.Db, r *http.Request, bean interface{}, id uint64) error {
	if t, ok := bean.(xorm.TableName); ok {
		query := builder.Select().From(t.TableName()).Where(builder.Eq{"id": id})
		if res, err := db.Query(query); err != nil {
			return err
		} else if len(res) == 0 {
			return &notFound{}
		}
	}

	if err := readJSON(r, bean); err != nil {
		return err
	}

	var isReplace bool
	if r.Method == http.MethodPut {
		isReplace = true
	}

	if err := db.Update(bean, id, isReplace); err != nil {
		return err
	}
	return nil
}
