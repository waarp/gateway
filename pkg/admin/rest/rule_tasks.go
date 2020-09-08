package rest

import (
	"encoding/json"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

// RuleTask is the JSON representation of a rule task in requests made to
// the REST interface.
type RuleTask struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// ToModel transforms the JSON task into its database equivalent.
func (i RuleTask) ToModel() *model.Task {
	return &model.Task{
		Type: i.Type,
		Args: i.Args,
	}
}

// FromRuleTasks transforms the given list of database tasks into its JSON
// equivalent.
func FromRuleTasks(ts []model.Task) []RuleTask {
	tasks := make([]RuleTask, len(ts))
	for i, task := range ts {
		tasks[i] = RuleTask{
			Type: task.Type,
			Args: task.Args,
		}
	}
	return tasks
}

func doListTasks(db *database.DB, rule *OutRule, ruleID uint64) error {
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

	rule.PreTasks = FromRuleTasks(preTasks)
	rule.PostTasks = FromRuleTasks(postTasks)
	rule.ErrorTasks = FromRuleTasks(errorTasks)
	return nil
}

func taskUpdateDelete(ses *database.Session, rule *UptRule, ruleID uint64,
	isReplace bool) error {

	if isReplace {
		if err := ses.Execute("DELETE FROM tasks WHERE rule_id=?", ruleID); err != nil {
			return err
		}
	} else {
		if len(rule.PreTasks) == 0 && rule.PreTasks != nil {
			if err := ses.Execute("DELETE FROM tasks WHERE rule_id=? AND chain=?",
				ruleID, model.ChainPre); err != nil {
				return err
			}
		}
		if len(rule.PostTasks) == 0 && rule.PostTasks == nil {
			if err := ses.Execute("DELETE FROM tasks WHERE rule_id=? AND chain=?",
				ruleID, model.ChainPost); err != nil {
				return err
			}
		}
		if len(rule.ErrorTasks) == 0 && rule.ErrorTasks == nil {
			if err := ses.Execute("DELETE FROM tasks WHERE rule_id=? AND chain=?",
				ruleID, model.ChainError); err != nil {
				return err
			}
		}
	}
	return nil
}

func doTaskUpdate(ses *database.Session, rule *UptRule, ruleID uint64,
	isReplace bool) error {

	if err := taskUpdateDelete(ses, rule, ruleID, isReplace); err != nil {
		return err
	}

	for rank, t := range rule.PreTasks {
		task := t.ToModel()
		task.RuleID = ruleID
		task.Chain = model.ChainPre
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, t := range rule.PostTasks {
		task := t.ToModel()
		task.RuleID = ruleID
		task.Chain = model.ChainPost
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, t := range rule.ErrorTasks {
		task := t.ToModel()
		task.RuleID = ruleID
		task.Chain = model.ChainError
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}

	return nil
}
