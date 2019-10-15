package admin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func addTransfer(logger *log.Logger, db *database.Db) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		trans := model.Transfer{}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		err = json.Unmarshal(body, &trans)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		if err = db.Create(&trans); err != nil {
			handleErrors(w, logger, err)
			return
		}
		newID := fmt.Sprint(trans.ID)
		w.Header().Set("Location", APIPath+TransfersPath+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func getTransfer(logger *log.Logger, db *database.Db) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "transfer")
		if err != nil {
			handleErrors(w, logger, &notFound{})
		}
		transfer := &model.Transfer{ID: id}

		if err := restGet(db, transfer); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err = writeJSON(w, transfer); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func listTransfers(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "start"

		validSorting := []string{"start"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		conditions := make([]builder.Cond, 0)

		conditions = append(conditions, builder.Eq{"owner": database.Owner})
		remotes := r.Form["remote"]
		if len(remotes) > 0 {
			conditions = append(conditions, builder.In("remote", remotes))
		}
		accounts := r.Form["account"]
		if len(accounts) > 0 {
			conditions = append(conditions, builder.In("account", accounts))
		}
		rules := r.Form["rule"]
		if len(rules) > 0 {
			conditions = append(conditions, builder.In("rule", rules))
		}
		statuses := r.Form["status"]
		if len(statuses) > 0 {
			conditions = append(conditions, builder.In("status", statuses))
		}
		// TODO parse as time.Time
		start := r.Form["start"]
		if len(start) > 0 {
			conditions = append(conditions, builder.Gt{"start": start[0]})
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
		}

		results := []model.Transfer{}
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.Transfer{"transfers": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}
