package admin

import (
	"net/http"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getPartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partner := &model.Partner{
			Name: mux.Vars(r)["partner"],
		}

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

		if err := readJSON(r, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}

		test := &model.Partner{Name: partner.Name}

		if err := restCreate(db, partner, test); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.Header().Set("Location", RestURI+PartnersURI+"/"+partner.Name)
		w.WriteHeader(http.StatusCreated)
	}
}

func deletePartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partner := &model.Partner{
			Name: mux.Vars(r)["partner"],
		}
		if err := restDelete(db, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updatePartner(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old := &model.Partner{
			Name: mux.Vars(r)["partner"],
		}

		partner := &model.Partner{}
		if r.Method == http.MethodPatch {
			if err := restGet(db, partner); err != nil {
				handleErrors(w, logger, err)
				return
			}
		}

		if err := readJSON(r, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}

		if err := restUpdate(db, old, partner); err != nil {
			handleErrors(w, logger, err)
			return
		}

		w.Header().Set("Location", RestURI+PartnersURI+"/"+partner.Name)
		w.WriteHeader(http.StatusCreated)
	}
}
