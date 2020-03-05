// Package rest contains the handlers of the gateway's REST API.
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/gorilla/mux"
)

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

func parseListFilters(r *http.Request, validOrders map[string]string) (*database.Filters, error) {
	filters := &database.Filters{
		Limit:  20,
		Offset: 0,
		Order:  validOrders["default"],
	}

	if limStr := r.FormValue("limit"); limStr != "" {
		lim, err := strconv.Atoi(limStr)
		if err != nil {
			return nil, &badRequest{msg: "'limit' must be an int"}
		}
		filters.Limit = lim
	}
	if offStr := r.FormValue("offset"); offStr != "" {
		off, err := strconv.Atoi(offStr)
		if err != nil {
			return nil, &badRequest{msg: "'offset' must be an int"}
		}
		filters.Offset = off
	}

	if sortStr := r.FormValue("sort"); sortStr != "" {
		sort, ok := validOrders[sortStr]
		if !ok {
			return nil, &badRequest{msg: fmt.Sprintf("'%s' is not a valid sort parameter", sortStr)}
		}
		filters.Order = sort
	}
	return filters, nil
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
		return 0, &notFound{}
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

func exist(acc database.Accessor, bean interface{}) error {
	if ok, err := acc.Exists(bean); err != nil {
		return err
	} else if !ok {
		return &notFound{}
	}
	return nil
}

func get(acc database.Accessor, bean interface{}) error {
	if err := acc.Get(bean); err != nil {
		if err == database.ErrNotFound {
			return &notFound{}
		}
		return err
	}
	return nil
}

func location(r *http.Request, id ...uint64) string {
	r.URL.RawQuery = ""
	r.URL.Fragment = ""
	if len(id) > 0 {
		if strings.HasSuffix(r.URL.String(), "/") {
			return fmt.Sprintf("%s%v", r.URL.String(), id[0])
		}
		return fmt.Sprintf("%s/%v", r.URL.String(), id[0])
	}
	return r.URL.String()
}
