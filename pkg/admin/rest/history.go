package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

// FromHistory transforms the given database history entry into its JSON equivalent.
func FromHistory(h *model.TransferHistory) *api.OutHistory {
	return &api.OutHistory{
		ID:             h.ID,
		RemoteID:       h.RemoteTransferID,
		IsServer:       h.IsServer,
		IsSend:         h.IsSend,
		Requester:      h.Account,
		Requested:      h.Agent,
		Protocol:       h.Protocol,
		SourceFilename: h.SourceFilename,
		DestFilename:   h.DestFilename,
		Rule:           h.Rule,
		Start:          h.Start,
		Stop:           h.Stop,
		Status:         h.Status,
		ErrorCode:      h.Error.Code,
		ErrorMsg:       h.Error.Details,
		Step:           h.Step,
		Progress:       h.Progress,
		TaskNumber:     h.TaskNumber,
	}
}

// FromHistories transforms the given list of database history entries into its
// JSON equivalent.
func FromHistories(hs []model.TransferHistory) []api.OutHistory {
	hist := make([]api.OutHistory, len(hs))
	for i, h := range hs {
		hist[i] = api.OutHistory{
			ID:             h.ID,
			IsServer:       h.IsServer,
			IsSend:         h.IsSend,
			Requester:      h.Account,
			Requested:      h.Agent,
			Protocol:       h.Protocol,
			SourceFilename: h.SourceFilename,
			DestFilename:   h.DestFilename,
			Rule:           h.Rule,
			Start:          h.Start,
			Stop:           h.Stop,
			Status:         h.Status,
			ErrorCode:      h.Error.Code,
			ErrorMsg:       h.Error.Details,
			Step:           h.Step,
			Progress:       h.Progress,
			TaskNumber:     h.TaskNumber,
		}
	}
	return hist
}

func getHist(r *http.Request, db *database.DB) (*model.TransferHistory, error) {
	val := mux.Vars(r)["history"]
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}
	history := &model.TransferHistory{ID: id}
	if err := db.Get(history); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("transfer %v not found", id)
		}
		return nil, err
	}
	return history, nil
}

func parseHistoryCond(r *http.Request, filters *database.Filters) error {
	conditions := make([]builder.Cond, 0)

	conditions = append(conditions, builder.Eq{"owner": database.Owner})

	accounts := r.Form["requester"]
	if len(accounts) > 0 {
		conditions = append(conditions, builder.In("account", accounts))
	}
	agents := r.Form["requested"]
	if len(agents) > 0 {
		conditions = append(conditions, builder.In("agent", agents))
	}
	rules := r.Form["rule"]
	if len(rules) > 0 {
		conditions = append(conditions, builder.In("rule", rules))
	}
	statuses := r.Form["status"]
	if len(statuses) > 0 {
		conditions = append(conditions, builder.In("status", statuses))
	}
	protocols := r.Form["protocol"]
	// Validate requested protocols
	for _, p := range protocols {
		if _, ok := config.ProtoConfigs[p]; !ok {
			return badRequest("'%s' is not a valid protocol", p)
		}
	}

	if len(protocols) > 0 {
		conditions = append(conditions, builder.In("protocol", protocols))
	}
	starts := r.Form["start"]
	if len(starts) > 0 {
		start, err := time.Parse(time.RFC3339, starts[0])
		if err != nil {
			return badRequest("'%s' is not a valid date", starts[0])
		}
		conditions = append(conditions, builder.Gte{"start": start.UTC()})
	}
	stops := r.Form["stop"]
	if len(stops) > 0 {
		stop, err := time.Parse(time.RFC3339, stops[0])
		if err != nil {
			return badRequest("'%s' is not a valid date", stops[0])
		}
		conditions = append(conditions, builder.Lte{"stop": stop.UTC()})
	}
	filters.Conditions = builder.And(conditions...)

	return nil
}

func getHistory(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getHist(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, FromHistory(result))
		handleError(w, logger, err)
	}
}

func listHistory(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default":    "start ASC",
		"id+":        "id ASC",
		"id-":        "id DESC",
		"requested+": "agent ASC",
		"requested-": "agent DESC",
		"requester+": "account ASC",
		"requester-": "account DESC",
		"rule+":      "rule ASC",
		"rule-":      "rule DESC",
		"status+":    "status ASC",
		"status-":    "status DESC",
		"start+":     "start ASC",
		"start-":     "start DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		filters, err := parseListFilters(r, validSorting)
		if handleError(w, logger, err) {
			return
		}
		if err := parseHistoryCond(r, filters); handleError(w, logger, err) {
			return
		}

		var results []model.TransferHistory
		if err := db.Select(&results, filters); handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutHistory{"history": FromHistories(results)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func retryTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		check, err := getHist(r, db)
		if handleError(w, logger, err) {
			return
		}

		date := time.Now().UTC()
		if dateStr := r.FormValue("date"); dateStr != "" {
			date, err = time.Parse(time.RFC3339, dateStr)
			if handleError(w, logger, err) {
				return
			}
		}

		if check.IsServer {
			handleError(w, logger, badRequest("only the client can retry a transfer"))
			return
		}

		trans, err := check.Restart(db, date)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Create(trans); handleError(w, logger, err) {
			return
		}

		r.URL.Path = "/api/transfers"
		w.Header().Set("Location", location(r.URL, fmt.Sprint(check.ID)))
		w.WriteHeader(http.StatusCreated)
	}
}
