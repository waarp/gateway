package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportRules(logger *log.Logger, db database.ReadAccess) ([]file.Rule, error) {
	var dbRules model.Rules
	if err := db.Select(&dbRules).Run(); err != nil {
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

		logger.Info("Export Rule %s\n", src.Name)

		Rule := file.Rule{
			Name:           src.Name,
			IsSend:         src.IsSend,
			Path:           src.Path,
			LocalDir:       src.LocalDir,
			RemoteDir:      src.RemoteDir,
			TmpLocalRcvDir: src.TmpLocalRcvDir,
			Accesses:       accs,
			Pre:            pre,
			Post:           post,
			Error:          errors,
		}
		res[i] = Rule
	}

	return res, nil
}

func exportRuleAccesses(db database.ReadAccess, ruleID int64) ([]string, error) {
	var dbAccs model.RuleAccesses
	if err := db.Select(&dbAccs).Where("rule_id=?", ruleID).Run(); err != nil {
		return nil, err
	}

	res := make([]string, len(dbAccs))

	for i, src := range dbAccs {
		switch {
		case src.RemoteAgentID.Valid:
			var agent model.RemoteAgent
			if err := db.Get(&agent, "id=?", src.RemoteAgentID.Int64).Run(); err != nil {
				return nil, err
			}

			res[i] = fmt.Sprintf("remote::%s", agent.Name)

		case src.RemoteAccountID.Valid:
			var account model.RemoteAccount
			if err := db.Get(&account, "id=?", src.RemoteAccountID.Int64).Run(); err != nil {
				return nil, err
			}

			var agent model.RemoteAgent
			if err := db.Get(&agent, "id=?", account.RemoteAgentID).Run(); err != nil {
				return nil, err
			}

			res[i] = fmt.Sprintf("remote::%s::%s", agent.Name, account.Login)

		case src.LocalAgentID.Valid:
			var agent model.LocalAgent
			if err := db.Get(&agent, "id=?", src.LocalAgentID.Int64).Run(); err != nil {
				return nil, err
			}

			res[i] = fmt.Sprintf("local::%s", agent.Name)

		case src.LocalAccountID.Valid:
			var account model.LocalAccount
			if err := db.Get(&account, "id=?", src.LocalAccountID.Int64).Run(); err != nil {
				return nil, err
			}

			var agent model.LocalAgent
			if err := db.Get(&agent, "id=?", account.LocalAgentID).Run(); err != nil {
				return nil, err
			}

			res[i] = fmt.Sprintf("local::%s::%s", agent.Name, account.Login)

		default:
			//nolint:goerr113 // too specific for a base error
			return nil, fmt.Errorf("rule access is missing a target")
		}
	}

	return res, nil
}

func exportRuleTasks(db database.ReadAccess, ruleID int64, chain string) ([]file.Task, error) {
	var dbTasks model.Tasks
	if err := db.Select(&dbTasks).Where("rule_id=? AND chain=?", ruleID, chain).
		OrderBy("rank", true).Run(); err != nil {
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
