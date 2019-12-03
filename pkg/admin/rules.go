package admin

import (
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func createRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rule := &model.Rule{}
		if err := readJSON(r, rule); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := db.Create(rule); err != nil {
			handleErrors(w, logger, err)
			return
		}

		id := strconv.FormatUint(rule.ID, 10)
		w.Header().Set("Location", APIPath+RulesPath+"/"+id)
		w.WriteHeader(http.StatusCreated)
	}
}

func getRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["rule"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}

		rule := &model.Rule{ID: id}
		if err := restGet(db, rule); err != nil {
			handleErrors(w, logger, err)
			return
		}

		if err := writeJSON(w, rule); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func listRules(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "name"
		validSorting := []string{"name"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		filters := &database.Filters{
			Limit:  limit,
			Offset: offset,
			Order:  order,
		}

		rules := []model.Rule{}
		if err := db.Select(&rules, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.Rule{"rules": rules}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["rule"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		rule := &model.Rule{ID: id}

		if err := restDelete(db, rule); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
