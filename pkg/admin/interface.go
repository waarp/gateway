package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func getInterface(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["interface"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		inter := &model.Interface{ID: id}

		if err := restGet(db, inter); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, inter); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func listInterfaces(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "name"
		validSorting := []string{"name", "port", "type"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		types := r.Form["type"]
		conditions := make([]builder.Cond, 0)
		if len(types) > 0 {
			for _, typ := range types {
				if !model.Exists(model.Types, typ) {
					handleErrors(w, logger, &badRequest{
						msg: fmt.Sprintf("'%s' is not a valid protocol", typ)})
				}
			}
			conditions = append(conditions, builder.In("type", types))
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
		}

		results := []model.Interface{}
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.Interface{"interfaces": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createInterface(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inter := &model.Interface{}

		if err := restCreate(db, r, inter); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(inter.ID, 10)
		w.Header().Set("Location", RestURI+InterfacesURI+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteInterface(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["interface"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		inter := &model.Interface{ID: id}

		if err := restDelete(db, inter); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateInterface(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["interface"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		oldPart := &model.Interface{ID: id}
		newPart := &model.Interface{ID: id}

		if err := restUpdate(db, r, oldPart, newPart); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", RestURI+InterfacesURI+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
