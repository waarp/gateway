package admin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

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

func getIDs(src []string) ([]uint64, error) {
	res := make([]uint64, len(src))
	for i, item := range src {
		id, err := strconv.ParseUint(item, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid ID", item)
		}
		res[i] = id
	}
	return res, nil
}

func makeTransfersConditions(form url.Values) ([]builder.Cond, error) {
	conditions := make([]builder.Cond, 0)

	conditions = append(conditions, builder.Eq{"owner": database.Owner})

	remotes := form["remote"]
	if len(remotes) > 0 {
		remoteIDs, err := getIDs(remotes)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, builder.In("remote_id", remoteIDs))
	}
	accounts := form["account"]
	if len(accounts) > 0 {
		accountIDs, err := getIDs(accounts)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, builder.In("account_id", accountIDs))
	}
	rules := form["rule"]
	if len(rules) > 0 {
		ruleIDs, err := getIDs(rules)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, builder.In("rule_id", ruleIDs))
	}
	statuses := form["status"]
	if len(statuses) > 0 {
		conditions = append(conditions, builder.In("status", statuses))
	}
	starts := form["start"]
	if len(starts) > 0 {
		start, err := time.Parse(time.RFC3339, starts[0])
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, builder.Gte{"start": start.UTC()})
	}
	return conditions, nil
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

		conditions, err := makeTransfersConditions(r.Form)
		if err != nil {
			handleErrors(w, logger, &badRequest{msg: err.Error()})
			return
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
