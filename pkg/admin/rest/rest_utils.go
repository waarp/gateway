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

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals // global var is used by design
var (
	str    = utils.String
	strPtr = utils.StringPtr
	cStr   = func(s *string) types.CypherText { return types.CypherText(str(s)) }
)

type order struct {
	col string
	asc bool
}

type orders map[string]order

func parseSelectQuery(r *http.Request, db *database.DB, validOrders orders,
	elem database.SelectBean,
) (*database.SelectQuery, error) {
	query := db.Select(elem)

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
			return nil, badRequest(fmt.Sprintf("'%s' is not a valid sort parameter", sortStr))
		}
	}

	query.OrderBy(orderBy.col, orderBy.asc)

	return query, nil
}

// handleError returns `true` if an error has been caught. It returns `false`
// if there is no error, and execution can continue.
func handleError(w http.ResponseWriter, logger *log.Logger, err error) bool {
	if err == nil {
		return false
	}

	var nf *notFoundError
	if errors.As(err, &nf) {
		http.Error(w, nf.Error(), http.StatusNotFound)

		return true
	}

	var dbNF *database.NotFoundError
	if errors.As(err, &dbNF) {
		http.Error(w, dbNF.Error(), http.StatusNotFound)

		return true
	}

	var br *badRequestError
	if errors.As(err, &br) {
		http.Error(w, br.Error(), http.StatusBadRequest)

		return true
	}

	var val *database.ValidationError
	if errors.As(err, &val) {
		http.Error(w, val.Error(), http.StatusBadRequest)

		return true
	}

	var fo *forbidden
	if errors.As(err, &fo) {
		http.Error(w, fo.Error(), http.StatusForbidden)

		return true
	}

	logger.Error("Unexpected error: %s", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)

	return true
}

func writeJSON(w http.ResponseWriter, bean interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	// encoder.SetIndent("", "  ") //TODO: uncomment once tests have been adapted

	if err := encoder.Encode(bean); err != nil {
		return fmt.Errorf("failed to write response JSON object: %w", err)
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
