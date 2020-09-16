// Package rest contains the handlers of the gateway's REST API.
package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

var str = utils.String
var strPtr = utils.StringPtr

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

// handleError returns `true` if an error has been caught. It returns `false`
// if there is no error, and execution can continue.
func handleError(w http.ResponseWriter, logger *log.Logger, err error) bool {
	if err == nil {
		return false
	}

	var nf *errNotFound
	if errors.As(err, &nf) {
		http.Error(w, nf.Error(), http.StatusNotFound)
		return true
	}
	var br *errBadRequest
	if errors.As(err, &br) {
		http.Error(w, br.Error(), http.StatusBadRequest)
		return true
	}
	var f *forbidden
	if errors.As(err, &f) {
		http.Error(w, f.Error(), http.StatusForbidden)
		return true
	}
	var inv *database.ValidationError
	if errors.As(err, &inv) {
		http.Error(w, inv.Error(), http.StatusBadRequest)
		return true
	}
	var inp *database.InputError
	if errors.As(err, &inp) {
		http.Error(w, inp.Error(), http.StatusBadRequest)
		return true
	}

	logger.Errorf("Unexpected error: %s", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return true
}

func writeJSON(w http.ResponseWriter, bean interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bean); err != nil {
		return fmt.Errorf("failed to write response JSON object: %s", err)
	}
	return nil
}

func readJSON(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return badRequest("malformed JSON object: %s", err)
	}
	return nil
}

func location(u *url.URL, name string) string {
	loc := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   path.Join(u.Path, name),
	}
	return loc.String()
}

func locationUpdate(u *url.URL, name string) string {
	loc := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   path.Join(path.Dir(u.Path), name),
	}
	return loc.String()
}
