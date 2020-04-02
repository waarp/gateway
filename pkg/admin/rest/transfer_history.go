package rest

import (
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
)

// OutHistory is the JSON representation of a history entry in responses sent by
// the REST interface.
type OutHistory struct {
	ID             uint64                  `json:"id"`
	IsServer       bool                    `json:"isServer"`
	IsSend         bool                    `json:"isSend"`
	Account        string                  `json:"account"`
	Remote         string                  `json:"remote"`
	Protocol       string                  `json:"protocol"`
	SourceFilename string                  `json:"sourceFilename"`
	DestFilename   string                  `json:"destFilename"`
	Rule           string                  `json:"rule"`
	Start          time.Time               `json:"start"`
	Stop           time.Time               `json:"stop"`
	Status         model.TransferStatus    `json:"status"`
	ErrorCode      model.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg       string                  `json:"errorMsg,omitempty"`
}

// FromHistory transforms the given database history entry into its JSON equivalent.
func FromHistory(h *model.TransferHistory) *OutHistory {
	return &OutHistory{
		ID:             h.ID,
		IsServer:       h.IsServer,
		IsSend:         h.IsSend,
		Account:        h.Account,
		Remote:         h.Remote,
		Protocol:       h.Protocol,
		SourceFilename: h.SourceFilename,
		DestFilename:   h.DestFilename,
		Rule:           h.Rule,
		Start:          h.Start,
		Stop:           h.Stop,
		Status:         h.Status,
		ErrorCode:      h.Error.Code,
		ErrorMsg:       h.Error.Details,
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
			Account:        h.Account,
			Remote:         h.Remote,
			Protocol:       h.Protocol,
			SourceFilename: h.SourceFilename,
			DestFilename:   h.DestFilename,
			Rule:           h.Rule,
			Start:          h.Start,
			Stop:           h.Stop,
			Status:         h.Status,
			ErrorCode:      h.Error.Code,
			ErrorMsg:       h.Error.Details,
		}
	}
	return hist
}

func parseHistoryCond(r *http.Request, filters *database.Filters) error {
	conditions := make([]builder.Cond, 0)

	conditions = append(conditions, builder.Eq{"owner": database.Owner})

	accounts := r.Form["account"]
	if len(accounts) > 0 {
		conditions = append(conditions, builder.In("account", accounts))
	}
	remotes := r.Form["remote"]
	if len(remotes) > 0 {
		conditions = append(conditions, builder.In("remote", remotes))
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
			return &badRequest{msg: fmt.Sprintf("'%s' is not a valid protocol", p)}
		}
	}

	if len(protocols) > 0 {
		conditions = append(conditions, builder.In("protocol", protocols))
	}
	starts := r.Form["start"]
	if len(starts) > 0 {
		start, err := time.Parse(time.RFC3339, starts[0])
		if err != nil {
			return &badRequest{msg: fmt.Sprintf("'%s' is not a valid date", starts[0])}
		}
		conditions = append(conditions, builder.Gte{"start": start.UTC()})
	}
	stops := r.Form["stop"]
	if len(stops) > 0 {
		stop, err := time.Parse(time.RFC3339, stops[0])
		if err != nil {
			return &badRequest{msg: fmt.Sprintf("'%s' is not a valid date", stops[0])}
		}
		conditions = append(conditions, builder.Lte{"stop": stop.UTC()})
	}
	filters.Conditions = builder.And(conditions...)

	return nil
}

func getHistory(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "history")
			if err != nil {
				return err
			}
			result := &model.TransferHistory{ID: id}

			if err := get(db, result); err != nil {
				return err
			}

			return writeJSON(w, FromHistory(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func listHistory(logger *log.Logger, db *database.Db) http.HandlerFunc {
	validSorting := map[string]string{
		"default":  "start ASC",
		"id+":      "id ASC",
		"id-":      "id DESC",
		"remote+":  "remote ASC",
		"remote-":  "remote DESC",
		"account+": "account ASC",
		"account-": "account DESC",
		"rule+":    "rule ASC",
		"rule-":    "rule DESC",
		"status+":  "status ASC",
		"status-":  "status DESC",
		"start+":   "start ASC",
		"start-":   "start DESC",
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
