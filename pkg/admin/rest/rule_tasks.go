package rest

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// taskToDB transforms the JSON task into its database equivalent.
func taskToDB(task *api.Task) *model.Task {
	return &model.Task{
		Type: task.Type,
		Args: task.Args,
	}
}

// FromRuleTasks transforms the given list of database tasks into its JSON
// equivalent.
func FromRuleTasks(ts []*model.Task) []*api.Task {
	tasks := make([]*api.Task, len(ts))
	for i, task := range ts {
		tasks[i] = &api.Task{
			Type: task.Type,
			Args: task.Args,
		}
	}

	return tasks
}

func doListTasks(db database.ReadAccess, rule *api.OutRule, ruleID int64) error {
	var preTasks model.Tasks
	if err := db.Select(&preTasks).Where("rule_id=? AND chain=?", ruleID,
		model.ChainPre).Run(); err != nil {
		return fmt.Errorf("failed to retrieve pre-tasks: %w", err)
	}

	var postTasks model.Tasks
	if err := db.Select(&postTasks).Where("rule_id=? AND chain=?", ruleID,
		model.ChainPost).Run(); err != nil {
		return fmt.Errorf("failed to retrieve post-tasks: %w", err)
	}

	var errorTasks model.Tasks
	if err := db.Select(&errorTasks).Where("rule_id=? AND chain=?", ruleID,
		model.ChainError).Run(); err != nil {
		return fmt.Errorf("failed to retrieve error-tasks: %w", err)
	}

	rule.PreTasks = FromRuleTasks(preTasks)
	rule.PostTasks = FromRuleTasks(postTasks)
	rule.ErrorTasks = FromRuleTasks(errorTasks)

	return nil
}

func taskUpdateDelete(ses *database.Session, rule *api.InRule, ruleID int64,
	isReplace bool,
) error {
	var task model.Task

	if isReplace {
		if err := ses.DeleteAll(&task).Where("rule_id=?", ruleID).Run(); err != nil {
			return fmt.Errorf("failed to purge tasks: %w", err)
		}

		return nil
	}

	if rule.PreTasks != nil {
		if err := ses.DeleteAll(&task).Where("rule_id=? AND chain=?", ruleID,
			model.ChainPre).Run(); err != nil {
			return fmt.Errorf("failed to purge pre-tasks: %w", err)
		}
	}

	if rule.PostTasks != nil {
		if err := ses.DeleteAll(&task).Where("rule_id=? AND chain=?", ruleID,
			model.ChainPost).Run(); err != nil {
			return fmt.Errorf("failed to purge post-tasks: %w", err)
		}
	}

	if rule.ErrorTasks != nil {
		if err := ses.DeleteAll(&task).Where("rule_id=? AND chain=?", ruleID,
			model.ChainError).Run(); err != nil {
			return fmt.Errorf("failed to purge error-tasks: %w", err)
		}
	}

	return nil
}

func doTaskUpdate(ses *database.Session, rule *api.InRule, ruleID int64,
	isReplace bool,
) error {
	if err := taskUpdateDelete(ses, rule, ruleID, isReplace); err != nil {
		return err
	}

	for rank, t := range rule.PreTasks {
		task := taskToDB(t)
		task.RuleID = ruleID
		task.Chain = model.ChainPre
		task.Rank = int8(rank)

		if err := ses.Insert(task).Run(); err != nil {
			return fmt.Errorf("failed to insert pre-task n°%d %q: %w",
				task.Rank, task.Type, err)
		}
	}

	for rank, t := range rule.PostTasks {
		task := taskToDB(t)
		task.RuleID = ruleID
		task.Chain = model.ChainPost
		task.Rank = int8(rank)

		if err := ses.Insert(task).Run(); err != nil {
			return fmt.Errorf("failed to insert post-task n°%d %q: %w",
				task.Rank, task.Type, err)
		}
	}

	for rank, t := range rule.ErrorTasks {
		task := taskToDB(t)
		task.RuleID = ruleID
		task.Chain = model.ChainError
		task.Rank = int8(rank)

		if err := ses.Insert(task).Run(); err != nil {
			return fmt.Errorf("failed to insert error-task n°%d %q: %w",
				task.Rank, task.Type, err)
		}
	}

	return nil
}
