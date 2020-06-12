// Package rest contains the handlers of the gateway's REST API.
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

type errBadRequest string

func (e errBadRequest) Error() string { return string(e) }

func badRequest(format string, args ...interface{}) errBadRequest {
	return errBadRequest(fmt.Sprintf(format, args...))
}

type forbidden struct {
	msg string
}

func (e *forbidden) Error() string {
	return e.msg
}

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

func notFound(format string, args ...interface{}) errNotFound {
	return errNotFound(fmt.Sprintf(format, args...))
}

func parseListFilters(r *http.Request, validOrders map[string]string) (*database.Filters, error) {
	filters := &database.Filters{
		Limit:  20,
		Offset: 0,
		Order:  validOrders["default"],
	}

	if limStr := r.FormValue("limit"); limStr != "" {
		lim, err := strconv.Atoi(limStr)
		if err != nil {
			return nil, badRequest("'limit' must be an int")
		}
		filters.Limit = lim
	}
	if offStr := r.FormValue("offset"); offStr != "" {
		off, err := strconv.Atoi(offStr)
		if err != nil {
			return nil, badRequest("'offset' must be an int")
		}
		filters.Offset = off
	}

	if sortStr := r.FormValue("sort"); sortStr != "" {
		sort, ok := validOrders[sortStr]
		if !ok {
			return nil, badRequest(fmt.Sprintf("'%s' is not a valid sort parameter", sortStr))
		}
		filters.Order = sort
	}
	return filters, nil
}

func handleErrors(w http.ResponseWriter, logger *log.Logger, err error) {
	switch err.(type) {
	case errNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	case errBadRequest:
		http.Error(w, err.Error(), http.StatusBadRequest)
	case *forbidden:
		http.Error(w, err.Error(), http.StatusForbidden)
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
		return badRequest(err.Error())
	}
	return nil
}

func location(r *http.Request, names ...string) string {
	r.URL.RawQuery = ""
	r.URL.Fragment = ""
	for _, name := range names {
		if name == "" {
			continue
		}
		if strings.HasSuffix(r.URL.String(), "/") {
			return fmt.Sprintf("%s%s", r.URL.String(), name)
		}
		return fmt.Sprintf("%s/%s", r.URL.String(), name)
	}
	return r.URL.String()
}

func locationUpdate(r *http.Request, names ...string) string {
	r.URL.Path = path.Dir(r.URL.Path)
	return location(r, names...)
}
