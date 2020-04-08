package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"

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
		SourceFile: i.SourcePath,
		DestFile:   i.DestPath,
		Start:      i.Start,
	}
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID           uint64                  `json:"id"`
	RuleID       uint64                  `json:"ruleID"`
	IsServer     bool                    `json:"isServer"`
	AgentID      uint64                  `json:"agentID"`
	AccountID    uint64                  `json:"accountID"`
	TrueFilepath string                  `json:"trueFilepath"`
	SourcePath   string                  `json:"sourcePath"`
	DestPath     string                  `json:"destPath"`
	Start        time.Time               `json:"startDate"`
	Status       model.TransferStatus    `json:"status"`
	Step         model.TransferStep      `json:"step,omitempty"`
	Progress     uint64                  `json:"progress,omitempty"`
	TaskNumber   uint64                  `json:"taskNumber,omitempty"`
	ErrorCode    model.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg     string                  `json:"errorMsg,omitempty"`
}

// FromTransfer transforms the given database transfer into its JSON equivalent.
func FromTransfer(t *model.Transfer) *OutTransfer {
	return &OutTransfer{
		ID:           t.ID,
		RuleID:       t.RuleID,
		IsServer:     t.IsServer,
		AgentID:      t.AgentID,
		AccountID:    t.AccountID,
		TrueFilepath: t.TrueFilepath,
		SourcePath:   t.SourceFile,
		DestPath:     t.DestFile,
		Start:        t.Start,
		Status:       t.Status,
		Step:         t.Step,
		Progress:     t.Progress,
		TaskNumber:   t.TaskNumber,
		ErrorCode:    t.Error.Code,
		ErrorMsg:     t.Error.Details,
	}
}

// FromTransfers transforms the given list of database transfers into its
// JSON equivalent.
func FromTransfers(ts []model.Transfer) []OutTransfer {
	transfers := make([]OutTransfer, len(ts))
	for i, trans := range ts {
		transfers[i] = OutTransfer{
			ID:           trans.ID,
			RuleID:       trans.RuleID,
			IsServer:     trans.IsServer,
			AgentID:      trans.AgentID,
			AccountID:    trans.AccountID,
			TrueFilepath: trans.TrueFilepath,
			SourcePath:   trans.SourceFile,
			DestPath:     trans.DestFile,
			Start:        trans.Start,
			Status:       trans.Status,
			Step:         trans.Step,
			Progress:     trans.Progress,
			TaskNumber:   trans.TaskNumber,
			ErrorCode:    trans.Error.Code,
			ErrorMsg:     trans.Error.Details,
		}
	}
	return transfers
}

func getTrans(r *http.Request, db *database.DB) (*model.Transfer, error) {
	val := mux.Vars(r)["transfer"]
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}
	transfer := &model.Transfer{ID: id}
	if err := db.Get(transfer); err != nil {
		if err == database.ErrNotFound {
			return nil, notFound("transfer %v not found", id)
		}
		return nil, err
	}
	return transfer, nil
}

func createTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
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

			w.Header().Set("Location", location(r, fmt.Sprint(trans.ID)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getTrans(r, db)
			if err != nil {
				return err
			}

			return writeJSON(w, FromTransfer(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func parseIDs(src []string) ([]uint64, error) {
	res := make([]uint64, len(src))
	for i, item := range src {
		id, err := strconv.ParseUint(item, 10, 64)
		if err != nil {
			return nil, badRequest("'%s' is not a valid ID", item)
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
		agentIDs, err := parseIDs(agents)
		if err != nil {
			return err
		}
		conditions = append(conditions, builder.In("agent_id", agentIDs))
	}
	accounts := r.Form["account"]
	if len(accounts) > 0 {
		accountIDs, err := parseIDs(accounts)
		if err != nil {
			return err
		}
		conditions = append(conditions, builder.In("account_id", accountIDs))
	}
	rules := r.Form["rule"]
	if len(rules) > 0 {
		ruleIDs, err := parseIDs(rules)
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
func listTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {
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

func pauseTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getTrans(r, db)
			if err != nil {
				return err
			}

			if err := db.Get(check); err != nil {
				return err
			}

			if check.Status == model.StatusPaused || check.Status == model.StatusInterrupted {
				return badRequest("cannot pause an already interrupted transfer")
			}

			if check.Status == model.StatusPlanned {
				check.Status = model.StatusPaused
				if err := check.Update(db); err != nil {
					return err
				}
			} else {
				pipeline.Signals.SendSignal(check.ID, model.SignalPause)
			}

			w.Header().Set("Location", location(r, fmt.Sprint(check.ID)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func cancelTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getTrans(r, db)
			if err != nil {
				return err
			}

			if err := db.Get(check); err != nil {
				return err
			}

			if check.Status != model.StatusRunning {
				check.Status = model.StatusCancelled
				if err := pipeline.ToHistory(db, logger, check); err != nil {
					return err
				}
			} else {
				pipeline.Signals.SendSignal(check.ID, model.SignalPause)
			}

			r.URL.Path = APIPath + HistoryPath
			w.Header().Set("Location", location(r, fmt.Sprint(check.ID)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func resumeTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getTrans(r, db)
			if err != nil {
				return err
			}

			if err := db.Get(check); err != nil {
				return err
			}

			if check.IsServer {
				return badRequest("only the client can restart a transfer")
			}

			if check.Status != model.StatusPaused && check.Status != model.StatusInterrupted {
				return badRequest("cannot resume an already running transfer")
			}

			agent := &model.RemoteAgent{ID: check.AgentID}
			if err := db.Get(agent); err != nil {
				return fmt.Errorf("failed to retreive partner: %s", err.Error())
			}
			if agent.Protocol == "sftp" {
				return badRequest("cannot restart an SFTP transfer")
			}

			check.Status = model.StatusPlanned
			if err := check.Update(db); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, fmt.Sprint(check.ID)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
