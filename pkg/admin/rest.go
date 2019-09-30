package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

const (
	// StatusURI is the access path to the status entry point
	StatusURI = "/status"
	// InterfacesURI is the access path to the interfaces entry point
	InterfacesURI = "/interfaces"
	// PartnersURI is the access path to the partners entry point
	PartnersURI = "/partners"
	// AccountsURI is the access path to the partner accounts entry point
	AccountsURI = "/accounts"
	// CertsURI is the access path to the account certificates entry point
	CertsURI = "/certificates"
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
func Authentication(logger *log.Logger, db *database.Db) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			invalidCredentials := func() {
				logger.Warning("Invalid credentials")
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			}

			logger.Debugf("Received %s on %s", r.Method, r.URL)

			login, pswd, ok := r.BasicAuth()
			if !ok {
				invalidCredentials()
				return
			}

			user := &model.User{
				Login: login,
			}
			if err := db.Get(user); err != nil {
				if err == database.ErrNotFound {
					invalidCredentials()
				} else {
					handleErrors(w, logger, err)
				}
				return
			}

			if err := bcrypt.CompareHashAndPassword(user.Password, []byte(pswd)); err != nil {
				if err == bcrypt.ErrMismatchedHashAndPassword {
					invalidCredentials()
				} else {
					handleErrors(w, logger, err)
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

// getStatus is called when an HTTP request is received on the StatusURI path.
func getStatus(logger *log.Logger, services map[string]service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var statuses = make(Statuses)
		for name, serv := range services {
			code, reason := serv.State().Get()
			statuses[name] = Status{
				State:  code.Name(),
				Reason: reason,
			}
		}

		if err := writeJSON(w, statuses); err != nil {
			handleErrors(w, logger, err)
		}

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

func restCreate(db *database.Db, r *http.Request, bean model.Validator) error {
	if err := readJSON(r, bean); err != nil {
		return err
	}

	if err := bean.Validate(db, true); err != nil {
		switch err.(type) {
		case model.ErrInvalid:
			return &badRequest{msg: err.Error()}
		default:
			return err
		}
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

func restUpdate(db *database.Db, r *http.Request, old, new model.Validator) error {
	if exist, err := db.Exists(old); err != nil {
		return err
	} else if !exist {
		return &notFound{}
	}

	if r.Method == http.MethodPatch {
		if err := restGet(db, new); err != nil {
			return err
		}
	}
	if err := readJSON(r, new); err != nil {
		return err
	}
	if err := new.Validate(db, false); err != nil {
		switch err.(type) {
		case model.ErrInvalid:
			return &badRequest{msg: err.Error()}
		default:
			return err
		}
	}

	if err := db.Update(old, new); err != nil {
		return err
	}
	return nil
}
