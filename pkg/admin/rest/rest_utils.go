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

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

type order struct {
	col string
	asc bool
}

type orders map[string]order

func parseSelectQuery(r *http.Request, db database.ReadAccess, validOrders orders,
	elem database.SelectBean,
) (*database.SelectQuery, error) {
	query := db.Select(elem).Eager()

	var err error

	limit := 20
	if limStr := r.FormValue("limit"); limStr != "" {
		limit, err = strconv.Atoi(limStr)
		if err != nil {
			return nil, badRequest("'limit' must be an int")
		}
	}

	offset := 0

	if offStr := r.FormValue("offset"); offStr != "" {
		offset, err = strconv.Atoi(offStr)
		if err != nil {
			return nil, badRequest("'offset' must be an int")
		}
	}

	query.Limit(limit, offset)

	orderBy := validOrders["default"]

	if sortStr := r.FormValue("sort"); sortStr != "" {
		var ok bool

		orderBy, ok = validOrders[sortStr]
		if !ok {
			return nil, badRequestf("%q is not a valid sort parameter", sortStr)
		}
	}

	query.OrderBy(orderBy.col, orderBy.asc)

	return query, nil
}

// HandleError returns `true` if an error has been caught. It returns `false`
// if there is no error, and execution can continue.
func HandleError(w http.ResponseWriter, logger *log.Logger, err error) bool {
	return handleError(w, logger, err)
}

func handleError(w http.ResponseWriter, logger *log.Logger, err error) bool {
	if err == nil {
		return false
	}

	if br, ok := errors.AsType[*badRequestError](err); ok {
		http.Error(w, br.Error(), http.StatusBadRequest)

		return true
	}

	if nf, ok := errors.AsType[*notFoundError](err); ok {
		http.Error(w, nf.Error(), http.StatusNotFound)

		return true
	}

	if fo, ok := errors.AsType[*forbidden](err); ok {
		http.Error(w, fo.Error(), http.StatusForbidden)

		return true
	}

	if val, ok := errors.AsType[*database.ForeignKeyError](err); ok {
		http.Error(w, val.Error(), http.StatusBadRequest)

		return true
	}

	if val, ok := errors.AsType[*database.ValidationError](err); ok {
		http.Error(w, val.Error(), http.StatusBadRequest)

		return true
	}

	if dbNF, ok := errors.AsType[*database.NotFoundError](err); ok {
		http.Error(w, dbNF.Error(), http.StatusNotFound)

		return true
	}

	logger.Errorf("Unexpected error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)

	return true
}

func WriteJSON(w http.ResponseWriter, obj any) error {
	return writeJSON(w, obj)
}

func ReadJSON(r *http.Request, dest any) error {
	return readJSON(r, dest)
}

func writeJSON(w http.ResponseWriter, bean any) error {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	// encoder.SetIndent("", "  ") //TODO: uncomment once tests have been adapted

	if err := encoder.Encode(bean); err != nil {
		return fmt.Errorf("failed to write response JSON object: %w", err)
	}

	return nil
}

func readJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return badRequestf("malformed JSON object: %v", err)
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

func LocationUpdate(u *url.URL, name string) string {
	return locationUpdate(u, name)
}

func GetURLVar(r *http.Request, name string) (string, bool) {
	val, ok := mux.Vars(r)[name]
	if !ok {
		return "", false
	}

	esc, err := url.PathUnescape(val)
	if err != nil {
		return val, true
	}

	return esc, true
}
