package admin

import (
	"net/http"
	"strconv"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getPartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["partner"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		partner := &model.Partner{ID: id}

		if err := restGet(db, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func listPartners(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "name"
		validSorting := []string{"address", "name", "port", "type"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		types := r.Form["type"]
		addresses := r.Form["address"]

		conditions := make([]builder.Cond, 0)
		if len(types) > 0 {
			conditions = append(conditions, builder.In("type", types))
		}
		if len(addresses) > 0 {
			conditions = append(conditions, builder.In("address", addresses))
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
		}

		results := &[]*model.Partner{}
		if err := db.Select(results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string]*[]*model.Partner{"partners": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createPartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partner := &model.Partner{}

		if err := restCreate(db, r, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(partner.ID, 10)
		w.Header().Set("Location", RestURI+PartnersURI+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func deletePartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["partner"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		partner := &model.Partner{ID: id}

		if err := restDelete(db, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updatePartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["partner"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		oldPart := &model.Partner{ID: id}
		newPart := &model.Partner{ID: id}

		if err := restUpdate(db, r, oldPart, newPart); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", RestURI+PartnersURI+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
