package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"github.com/gorilla/mux"
)

// FromHistory transforms the given database history entry into its JSON equivalent.
func FromHistory(h *model.TransferHistory) *api.OutHistory {
	var stop *time.Time
	if !h.Stop.IsZero() {
		stop = &h.Stop
	}

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
		Start:          h.Start.Local(),
		Stop:           stop,
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
	for i := range hs {
		hist[i] = *FromHistory(&hs[i])
	}
	return hist
}

func getHist(r *http.Request, db *database.DB) (*model.TransferHistory, error) {
	val := mux.Vars(r)["history"]
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}
	var history model.TransferHistory
	if err := db.Get(&history, "id=?", id).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("transfer %v not found", id)
		}
		return nil, err
	}
	return &history, nil
}

func parseHistoryCond(r *http.Request, query *database.SelectQuery) error {
	accounts := r.Form["requester"]
	if len(accounts) > 0 {
		query.In("account", accounts)
	}
	agents := r.Form["requested"]
	if len(agents) > 0 {
		query.In("agent", agents)
	}
	rules := r.Form["rule"]
	if len(rules) > 0 {
		query.In("rule", rules)
	}
	statuses := r.Form["status"]
	if len(statuses) > 0 {
		query.In("status", statuses)
	}
	protocols := r.Form["protocol"]
	// Validate requested protocols
	for _, p := range protocols {
		if _, ok := config.ProtoConfigs[p]; !ok {
			return badRequest("'%s' is not a valid protocol", p)
		}
	}

	if len(protocols) > 0 {
		query.In("protocol", protocols)
	}
	starts := r.Form["start"]
	if len(starts) > 0 {
		start, err := time.Parse(time.RFC3339Nano, starts[0])
		if err != nil {
			return badRequest("'%s' is not a valid date", starts[0])
		}
		query.Where("start >= ?", start.UTC().Truncate(time.Microsecond).
			Format(time.RFC3339Nano))
	}
	stops := r.Form["stop"]
	if len(stops) > 0 {
		stop, err := time.Parse(time.RFC3339Nano, stops[0])
		if err != nil {
			return badRequest("'%s' is not a valid date", stops[0])
		}
		query.Where("stop <= ?", stop.UTC().Truncate(time.Microsecond).
			Format(time.RFC3339Nano))
	}

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
	validSorting := orders{
		"default":    order{col: "start", asc: true},
		"id+":        order{col: "id", asc: true},
		"id-":        order{col: "id", asc: false},
		"requested+": order{col: "agent", asc: true},
		"requested-": order{col: "agent", asc: false},
		"requester+": order{col: "account", asc: true},
		"requester-": order{col: "account", asc: false},
		"rule+":      order{col: "rule", asc: true},
		"rule-":      order{col: "rule", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
		"start+":     order{col: "start", asc: true},
		"start-":     order{col: "start", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var results model.Histories
		query, err := parseSelectQuery(r, db, validSorting, &results)
		if handleError(w, logger, err) {
			return
		}
		if err := parseHistoryCond(r, query); handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
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
			date, err = time.Parse(time.RFC3339Nano, dateStr)
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

		if err := db.Insert(trans).Run(); handleError(w, logger, err) {
			return
		}

		r.URL.Path = "/api/transfers"
		w.Header().Set("Location", location(r.URL, fmt.Sprint(check.ID)))
		w.WriteHeader(http.StatusCreated)
	}
}
