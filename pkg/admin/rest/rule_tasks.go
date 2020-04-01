package rest

import (
	"encoding/json"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

// InRuleTask is the JSON representation of a rule task in requests made to
// the REST interface.
type InRuleTask struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// ToModel transforms the JSON task into its database equivalent.
func (i InRuleTask) ToModel() *model.Task {
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

// FromRuleTasks transforms the given list of database tasks into its JSON
// equivalent.
func FromRuleTasks(ts []model.Task) []OutRuleTask {
	tasks := make([]OutRuleTask, len(ts))
	for i, task := range ts {
		tasks[i] = OutRuleTask{
			Type: task.Type,
			Args: task.Args,
		}
	}
	return tasks
}

func listTasks(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			rule, err := getRl(r, db)
			if err != nil {
				return err
			}

			preTasks := []model.Task{}
			preFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": rule.ID, "chain": model.ChainPre},
			}
			if err := db.Select(&preTasks, preFilters); err != nil {
				return err
			}

			postTasks := []model.Task{}
			postFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": rule.ID, "chain": model.ChainPost},
			}
			if err := db.Select(&postTasks, postFilters); err != nil {
				return err
			}

			errorTasks := []model.Task{}
			errorFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": rule.ID, "chain": model.ChainError},
			}
			if err := db.Select(&errorTasks, errorFilters); err != nil {
				return err
			}

			res := map[string][]OutRuleTask{
				"preTasks":   FromRuleTasks(preTasks),
				"postTasks":  FromRuleTasks(postTasks),
				"errorTasks": FromRuleTasks(errorTasks),
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
		task := t.ToModel()
		task.RuleID = ruleID
		task.Chain = model.ChainPre
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, t := range req["postTasks"] {
		task := t.ToModel()
		task.RuleID = ruleID
		task.Chain = model.ChainPost
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, t := range req["errorTasks"] {
		task := t.ToModel()
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

func updateTasks(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			rule, err := getRl(r, db)
			if err != nil {
				return err
			}

			req := map[string][]InRuleTask{}
			if err := readJSON(r, &req); err != nil {
				return err
			}

			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			if err := doTaskUpdate(ses, req, rule.ID); err != nil {
				ses.Rollback()
				return err
			}

			w.Header().Set("Location", location2(r))
			w.WriteHeader(http.StatusCreated)

			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
		}
	}
}
