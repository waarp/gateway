package rest

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// FromHistory transforms the given database history entry into its JSON equivalent.
func FromHistory(db *database.DB, hist *model.HistoryEntry) (*api.OutHistory, error) {
	var stop *time.Time
	if !hist.Stop.IsZero() {
		stop = &hist.Stop
	}

	src := path.Base(hist.RemotePath)
	dst := filepath.Base(hist.LocalPath)

	if hist.IsSend {
		dst = path.Base(hist.RemotePath)
		src = filepath.Base(hist.LocalPath)
	}

	info, err := hist.GetTransferInfo(db)
	if err != nil {
		return nil, err
	}

	return &api.OutHistory{
		ID:             hist.ID,
		RemoteID:       hist.RemoteTransferID,
		IsServer:       hist.IsServer,
		IsSend:         hist.IsSend,
		Requester:      hist.Account,
		Requested:      hist.Agent,
		Protocol:       hist.Protocol,
		LocalFilepath:  hist.LocalPath,
		RemoteFilepath: hist.RemotePath,
		Filesize:       hist.Filesize,
		Rule:           hist.Rule,
		Start:          hist.Start.Local(),
		Stop:           stop,
		TransferInfo:   info,
		Status:         hist.Status,
		ErrorCode:      hist.Error.Code,
		ErrorMsg:       hist.Error.Details,
		Step:           hist.Step,
		Progress:       hist.Progress,
		TaskNumber:     hist.TaskNumber,
		SourceFilename: utils.NormalizePath(src),
		DestFilename:   utils.NormalizePath(dst),
	}, nil
}

// FromHistories transforms the given list of database history entries into its
// JSON equivalent.
func FromHistories(db *database.DB, hs []model.HistoryEntry) ([]api.OutHistory, error) {
	hist := make([]api.OutHistory, len(hs))

	for i := range hs {
		jHist, err := FromHistory(db, &hs[i])
		if err != nil {
			return nil, err
		}

		hist[i] = *jHist
	}

	return hist, nil
}

//nolint:dupl // duplicated code is about a different type
func getHist(r *http.Request, db *database.DB) (*model.HistoryEntry, error) {
	val := mux.Vars(r)["history"]

	id, err := strconv.ParseUint(val, 10, 64) //nolint:gomnd // no need for a constant here
	if err != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}

	var history model.HistoryEntry
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

	if agents := r.Form["requested"]; len(agents) > 0 {
		query.In("agent", agents)
	}

	if rules := r.Form["rule"]; len(rules) > 0 {
		query.In("rule", rules)
	}

	if statuses := r.Form["status"]; len(statuses) > 0 {
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

		hist, err := FromHistory(db, result)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, hist))
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
		"stop+":      order{col: "stop", asc: true},
		"stop-":      order{col: "stop", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var results model.HistoryEntries
		query, err := parseSelectQuery(r, db, validSorting, &results)

		if handleError(w, logger, err) {
			return
		}

		if err2 := parseHistoryCond(r, query); handleError(w, logger, err2) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		hist, err := FromHistories(db, results)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutHistory{"history": hist}
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
