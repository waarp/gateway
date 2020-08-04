package backup

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func exportRules(logger *log.Logger, db *database.Session) ([]rule, error) {
	dbRules := []model.Rule{}
	if err := db.Select(&dbRules, nil); err != nil {
		return nil, err
	}
	res := make([]rule, len(dbRules))

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

		logger.Infof("Export rule %s\n", src.Name)
		rule := rule{
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
		res[i] = rule
	}
	return res, nil
}

func exportRuleAccesses(db *database.Session, ruleID uint64) ([]string, error) {
	dbAccs := []model.RuleAccess{}
	filters := &database.Filters{
		Conditions: builder.Eq{"rule_id": ruleID},
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

func exportRuleTasks(db *database.Session, ruleID uint64, chain string) ([]ruleTask, error) {
	dbTasks := []model.Task{}
	filters := &database.Filters{
		Conditions: builder.And(
			builder.Eq{"rule_id": ruleID},
			builder.Eq{"chain": chain},
		),
		Order: "rank ASC",
	}
	if err := db.Select(&dbTasks, filters); err != nil {
		return nil, err
	}
	res := make([]ruleTask, len(dbTasks))

	for i, src := range dbTasks {
		res[i] = ruleTask{
			Type: src.Type,
			Args: src.Args,
		}
	}
	return res, nil
}
