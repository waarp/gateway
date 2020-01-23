package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InTransfer is the JSON representation of a transfer in requests made to
// the REST interface.
type InTransfer struct {
	RuleID     uint64    `json:"ruleID"`
	IsServer   bool      `json:"isServer"`
	AgentID    uint64    `json:"agentID"`
	AccountID  uint64    `json:"accountID"`
	SourcePath string    `json:"sourcePath"`
	DestPath   string    `json:"destPath"`
	Start      time.Time `json:"startDate"`
}

// ToModel transforms the JSON transfer into its database equivalent.
func (i *InTransfer) ToModel() *model.Transfer {
	return &model.Transfer{
		RuleID:     i.RuleID,
		IsServer:   i.IsServer,
		AgentID:    i.AgentID,
		AccountID:  i.AccountID,
		SourcePath: i.SourcePath,
		DestPath:   i.DestPath,
		Start:      i.Start,
	}
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID         uint64                  `json:"id"`
	RuleID     uint64                  `json:"ruleID"`
	IsServer   bool                    `json:"isServer"`
	AgentID    uint64                  `json:"agentID"`
	AccountID  uint64                  `json:"accountID"`
	SourcePath string                  `json:"sourcePath"`
	DestPath   string                  `json:"destPath"`
	Start      time.Time               `json:"startDate"`
	Status     model.TransferStatus    `json:"status"`
	Progress   uint64                  `json:"progress"`
	TaskNumber uint64                  `json:"taskNumber"`
	ErrorCode  model.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg   string                  `json:"errorMsg,omitempty"`
}

// FromTransfer transforms the given database transfer into its JSON equivalent.
func FromTransfer(t *model.Transfer) *OutTransfer {
	return &OutTransfer{
		ID:         t.ID,
		RuleID:     t.RuleID,
		IsServer:   t.IsServer,
		AgentID:    t.AgentID,
		AccountID:  t.AccountID,
		SourcePath: t.SourcePath,
		DestPath:   t.DestPath,
		Start:      t.Start,
		Status:     t.Status,
		Progress:   t.Progress,
		TaskNumber: t.TaskNumber,
		ErrorCode:  t.Error.Code,
		ErrorMsg:   t.Error.Details,
	}
}

// FromTransfers transforms the given list of database transfers into its
// JSON equivalent.
func FromTransfers(ts []model.Transfer) []OutTransfer {
	transfers := make([]OutTransfer, len(ts))
	for i, trans := range ts {
		transfers[i] = OutTransfer{
			ID:         trans.ID,
			RuleID:     trans.RuleID,
			IsServer:   trans.IsServer,
			AgentID:    trans.AgentID,
			AccountID:  trans.AccountID,
			SourcePath: trans.SourcePath,
			DestPath:   trans.DestPath,
			Start:      trans.Start,
			Status:     trans.Status,
			ErrorCode:  trans.Error.Code,
			ErrorMsg:   trans.Error.Details,
		}
	}
	return transfers
}

func createTransfer(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonTrans := &InTransfer{}
			if err := readJSON(r, jsonTrans); err != nil {
				return err
			}

			trans := jsonTrans.ToModel()
			if err := db.Create(trans); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, trans.ID))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getTransfer(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "transfer")
			if err != nil {
				return err
			}
			result := &model.Transfer{ID: id}

			if err := get(db, result); err != nil {
				return err
			}

			return writeJSON(w, FromTransfer(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getIDs(src []string) ([]uint64, error) {
	res := make([]uint64, len(src))
	for i, item := range src {
		id, err := strconv.ParseUint(item, 10, 64)
		if err != nil {
			return nil, &badRequest{msg: fmt.Sprintf("'%s' is not a valid ID", item)}
		}
		res[i] = id
	}
	return res, nil
}

func parseTransferCond(r *http.Request, filters *database.Filters) error {
	conditions := make([]builder.Cond, 0)

	conditions = append(conditions, builder.Eq{"owner": database.Owner})

	agents := r.Form["agent"]
	if len(agents) > 0 {
		agentIDs, err := getIDs(agents)
		if err != nil {
			return err
		}
		conditions = append(conditions, builder.In("agent_id", agentIDs))
	}
	accounts := r.Form["account"]
	if len(accounts) > 0 {
		accountIDs, err := getIDs(accounts)
		if err != nil {
			return err
		}
		conditions = append(conditions, builder.In("account_id", accountIDs))
	}
	rules := r.Form["rule"]
	if len(rules) > 0 {
		ruleIDs, err := getIDs(rules)
		if err != nil {
			return err
		}
		conditions = append(conditions, builder.In("rule_id", ruleIDs))
	}
	statuses := r.Form["status"]
	if len(statuses) > 0 {
		conditions = append(conditions, builder.In("status", statuses))
	}
	starts := r.Form["start"]
	if len(starts) > 0 {
		start, err := time.Parse(time.RFC3339, starts[0])
		if err != nil {
			return err
		}
		conditions = append(conditions, builder.Gte{"start": start.UTC()})
	}
	filters.Conditions = builder.And(conditions...)

	return nil
}

//nolint:dupl
func listTransfers(logger *log.Logger, db *database.Db) http.HandlerFunc {
	validSorting := map[string]string{
		"default":  "id ASC",
		"id+":      "id ASC",
		"id-":      "id DESC",
		"remote+":  "remote_id ASC",
		"remote-":  "remote_id DESC",
		"account+": "account_id ASC",
		"account-": "account_id DESC",
		"rule+":    "rule_id ASC",
		"rule-":    "rule_id DESC",
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
			if err := parseTransferCond(r, filters); err != nil {
				return err
			}

			var results []model.Transfer
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutTransfer{"transfers": FromTransfers(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
