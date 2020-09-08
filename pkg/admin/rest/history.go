package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

// OutHistory is the JSON representation of a history entry in responses sent by
// the REST interface.
type OutHistory struct {
	ID             uint64                  `json:"id"`
	IsServer       bool                    `json:"isServer"`
	IsSend         bool                    `json:"isSend"`
	Requester      string                  `json:"requester"`
	Requested      string                  `json:"requested"`
	Protocol       string                  `json:"protocol"`
	SourceFilename string                  `json:"sourceFilename"`
	DestFilename   string                  `json:"destFilename"`
	Rule           string                  `json:"rule"`
	Start          time.Time               `json:"start"`
	Stop           time.Time               `json:"stop"`
	Status         model.TransferStatus    `json:"status"`
	ErrorCode      model.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg       string                  `json:"errorMsg,omitempty"`
	Step           model.TransferStep      `json:"step,omitempty"`
	Progress       uint64                  `json:"progress,omitempty"`
	TaskNumber     uint64                  `json:"taskNumber,omitempty"`
}

// FromHistory transforms the given database history entry into its JSON equivalent.
func FromHistory(h *model.TransferHistory) *OutHistory {
	return &OutHistory{
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

// FromHistories transforms the given list of database history entries into its
// JSON equivalent.
func FromHistories(hs []model.TransferHistory) []OutHistory {
	hist := make([]OutHistory, len(hs))
	for i, h := range hs {
		hist[i] = OutHistory{
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
		if err == database.ErrNotFound {
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
		err := func() error {
			result, err := getHist(r, db)
			if err != nil {
				return err
			}

			return writeJSON(w, FromHistory(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
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
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			if err := parseHistoryCond(r, filters); err != nil {
				return err
			}

			var results []model.TransferHistory
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutHistory{"history": FromHistories(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func retryTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getHist(r, db)
			if err != nil {
				return err
			}

			date := time.Now().UTC()
			if dateStr := r.FormValue("date"); dateStr != "" {
				date, err = time.Parse(time.RFC3339, dateStr)
				if err != nil {
					return err
				}
			}

			if check.IsServer {
				return badRequest("only the client can retry a transfer")
			}

			if check.Status == model.StatusDone {
				return badRequest("cannot retry a finished transfer")
			}

			trans, err := check.Restart(db, date)
			if err != nil {
				return err
			}

			if err := db.Create(trans); err != nil {
				return err
			}

			r.URL.Path = APIPath + TransfersPath
			w.Header().Set("Location", location(r, fmt.Sprint(check.ID)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
