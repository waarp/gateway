package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func getHistory(logger *log.Logger, db *database.Db) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "history")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}

		history := &model.TransferHistory{ID: id}

		if err = restGet(db, history); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err = writeJSON(w, history); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func makeHistoryConditions(form url.Values) ([]builder.Cond, error) {
	conditions := make([]builder.Cond, 0)

	conditions = append(conditions, builder.Eq{"owner": database.Owner})

	accounts := form["account"]
	if len(accounts) > 0 {
		conditions = append(conditions, builder.In("account", accounts))
	}
	remotes := form["remote"]
	if len(remotes) > 0 {
		conditions = append(conditions, builder.In("remote", remotes))
	}
	rules := form["rule"]
	if len(rules) > 0 {
		conditions = append(conditions, builder.In("rule", rules))
	}
	statuses := form["status"]
	if len(statuses) > 0 {
		conditions = append(conditions, builder.In("status", statuses))
	}
	protocols := form["protocol"]
	// Validate requested protocols
	for _, p := range protocols {
		if !model.IsValidProtocol(p) {
			return nil, fmt.Errorf("%s is not a valid protocol", p)
		}
	}

	if len(protocols) > 0 {
		conditions = append(conditions, builder.In("protocol", protocols))
	}
	starts := form["start"]
	if len(starts) > 0 {
		start, err := time.Parse(time.RFC3339, starts[0])
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, builder.Gte{"start": start.UTC()})
	}
	stops := form["stop"]
	if len(stops) > 0 {
		stop, err := time.Parse(time.RFC3339, stops[0])
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, builder.Lte{"stop": stop.UTC()})
	}
	return conditions, nil
}

func makeHistoryFilters(r *http.Request) (*database.Filters, error) {
	limit := 20
	offset := 0
	order := "start"

	validSorting := []string{"account", "remote", "rule", "status", "protocol", "start"}

	if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
		return nil, err
	}

	conditions, err := makeHistoryConditions(r.Form)
	if err != nil {
		return nil, err
	}

	filters := &database.Filters{
		Limit:      limit,
		Offset:     offset,
		Order:      order,
		Conditions: builder.And(conditions...),
	}
	return filters, nil
}

func listHistory(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filters, err := makeHistoryFilters(r)
		if err != nil {
			handleErrors(w, logger, &badRequest{msg: err.Error()})
			return
		}
		results := []model.TransferHistory{}
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.TransferHistory{"history": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}
