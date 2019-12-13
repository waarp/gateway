package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

// InRuleTask is the JSON representation of a rule task in requests made to
// the REST interface.
type InRuleTask struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

func (i InRuleTask) toModel() *model.Task {
	return &model.Task{
		Type: i.Type,
		Args: i.Args,
	}
}

// OutRuleTask is the JSON representation of a rule task in responses sent by
// the REST interface.
type OutRuleTask struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

func fromRuleTasks(ts []model.Task) []OutRuleTask {
	tasks := make([]OutRuleTask, len(ts))
	for i, task := range ts {
		tasks[i] = OutRuleTask{
			Type: task.Type,
			Args: json.RawMessage(task.Args),
		}
	}
	return tasks
}

func listTasks(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := parseID(r, "rule")
			if err != nil {
				return err
			}

			if ok, err := db.Exists(&model.Rule{ID: ruleID}); err != nil {
				return err
			} else if !ok {
				return &notFound{}
			}

			preTasks := []model.Task{}
			preFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": ruleID, "chain": model.ChainPre},
			}
			if err := db.Select(&preTasks, preFilters); err != nil {
				return err
			}

			postTasks := []model.Task{}
			postFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": ruleID, "chain": model.ChainPost},
			}
			if err := db.Select(&postTasks, postFilters); err != nil {
				return err
			}

			errorTasks := []model.Task{}
			errorFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": ruleID, "chain": model.ChainError},
			}
			if err := db.Select(&errorTasks, errorFilters); err != nil {
				return err
			}

			res := map[string][]OutRuleTask{
				"preTasks":   fromRuleTasks(preTasks),
				"postTasks":  fromRuleTasks(postTasks),
				"errorTasks": fromRuleTasks(errorTasks),
			}

			if err := writeJSON(w, res); err != nil {
				return err
			}

			w.WriteHeader(http.StatusOK)
			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
		}
	}
}

func doTaskUpdate(ses *database.Session, req map[string][]InRuleTask, ruleID uint64) error {
	if err := ses.Execute("DELETE FROM tasks WHERE rule_id=?", ruleID); err != nil {
		return err
	}
	for rank, t := range req["preTasks"] {
		task := t.toModel()
		task.RuleID = ruleID
		task.Chain = model.ChainPre
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, t := range req["postTasks"] {
		task := t.toModel()
		task.RuleID = ruleID
		task.Chain = model.ChainPost
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, t := range req["errorTasks"] {
		task := t.toModel()
		task.RuleID = ruleID
		task.Chain = model.ChainError
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	if err := ses.Commit(); err != nil {
		return err
	}

	return nil
}

func updateTasks(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := strconv.ParseUint(mux.Vars(r)["rule"], 10, 64)
			if err != nil {
				return &notFound{}
			}

			if ok, err := db.Exists(&model.Rule{ID: ruleID}); err != nil {
				return err
			} else if !ok {
				return &notFound{}
			}

			req := map[string][]InRuleTask{}
			if err := readJSON(r, &req); err != nil {
				return err
			}

			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			if err := doTaskUpdate(ses, req, ruleID); err != nil {
				ses.Rollback()
				return err
			}

			w.Header().Set("Location", location(r))
			w.WriteHeader(http.StatusCreated)

			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
		}
	}
}
