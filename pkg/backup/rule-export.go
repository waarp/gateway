package backup

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportRules(logger *log.Logger, db *database.Session) ([]file.Rule, error) {
	var dbRules []model.Rule
	if err := db.Select(&dbRules, nil); err != nil {
		return nil, err
	}
	res := make([]file.Rule, len(dbRules))

	for i, src := range dbRules {

		accs, err := exportRuleAccesses(db, src.ID)
		if err != nil {
			return nil, err
		}
		pre, err := exportRuleTasks(db, src.ID, "PRE")
		if err != nil {
			return nil, err
		}
		post, err := exportRuleTasks(db, src.ID, "POST")
		if err != nil {
			return nil, err
		}
		errors, err := exportRuleTasks(db, src.ID, "ERROR")
		if err != nil {
			return nil, err
		}

		logger.Infof("Export Rule %s\n", src.Name)
		Rule := file.Rule{
			Name:     src.Name,
			IsSend:   src.IsSend,
			Path:     src.Path,
			InPath:   src.InPath,
			OutPath:  src.OutPath,
			WorkPath: src.WorkPath,
			Accesses: accs,
			Pre:      pre,
			Post:     post,
			Error:    errors,
		}
		res[i] = Rule
	}
	return res, nil
}

func exportRuleAccesses(db *database.Session, RuleID uint64) ([]string, error) {
	var dbAccs []model.RuleAccess
	filters := &database.Filters{
		Conditions: builder.Eq{"Rule_id": RuleID},
	}
	if err := db.Select(&dbAccs, filters); err != nil {
		return nil, err
	}
	res := make([]string, len(dbAccs))

	for i, src := range dbAccs {
		if src.ObjectType == "remote_agents" {
			agent := &model.RemoteAgent{
				ID: src.ObjectID,
			}
			if err := db.Get(agent); err != nil {
				return nil, err
			}
			res[i] = fmt.Sprintf("remote::%s", agent.Name)
		} else if src.ObjectType == "remote_accounts" {
			account := &model.RemoteAccount{
				ID: src.ObjectID,
			}
			if err := db.Get(account); err != nil {
				return nil, err
			}
			agent := &model.RemoteAgent{
				ID: account.RemoteAgentID,
			}
			if err := db.Get(agent); err != nil {
				return nil, err
			}
			res[i] = fmt.Sprintf("remote::%s::%s", agent.Name, account.Login)
		} else if src.ObjectType == "local_agents" {
			agent := &model.LocalAgent{
				ID: src.ObjectID,
			}
			if err := db.Get(agent); err != nil {
				return nil, err
			}
			res[i] = fmt.Sprintf("local::%s", agent.Name)
		} else if src.ObjectType == "local_accounts" {
			account := &model.LocalAccount{
				ID: src.ObjectID,
			}
			if err := db.Get(account); err != nil {
				return nil, err
			}
			agent := &model.LocalAgent{
				ID: account.LocalAgentID,
			}
			if err := db.Get(agent); err != nil {
				return nil, err
			}
			res[i] = fmt.Sprintf("local::%s::%s", agent.Name, account.Login)
		}
	}
	return res, nil
}

func exportRuleTasks(db *database.Session, RuleID uint64, chain string) ([]file.Task, error) {
	var dbTasks []model.Task
	filters := &database.Filters{
		Conditions: builder.And(
			builder.Eq{"Rule_id": RuleID},
			builder.Eq{"chain": chain},
		),
		Order: "rank ASC",
	}
	if err := db.Select(&dbTasks, filters); err != nil {
		return nil, err
	}
	res := make([]file.Task, len(dbTasks))

	for i, src := range dbTasks {
		res[i] = file.Task{
			Type: src.Type,
			Args: src.Args,
		}
	}
	return res, nil
}
