package backup

import (
	"fmt"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importRules(db *database.Session, list []rule) error {

	for _, src := range list {

		//  Create model with basic info to check existence
		rule := &model.Rule{
			Name:   src.Name,
			IsSend: src.IsSend,
		}

		// Check if rule exists
		exists := true
		err := db.Get(rule)
		if err != nil {
			if err == database.ErrNotFound {
				exists = false
			} else {
				return err
			}
		}

		// Populate
		rule.Path = src.Path
		rule.InPath = src.InPath
		rule.OutPath = src.OutPath
		rule.WorkPath = src.WorkPath

		// Create/Update
		if exists {
			fmt.Printf("Update rule %s\n", rule.Name)
			err = db.Update(rule, rule.ID, false)
		} else {
			fmt.Printf("Create rule %s\n", rule.Name)
			err = db.Create(rule)
		}
		if err != nil {
			return err
		}
		if err = importRuleAccesses(db, src.Accesses, rule.ID); err != nil {
			return err
		}
		if err = importRuleTasks(db, src.Pre, rule.ID, model.ChainPre); err != nil {
			return err
		}
		if err = importRuleTasks(db, src.Post, rule.ID, model.ChainPost); err != nil {
			return err
		}
		if err = importRuleTasks(db, src.Error, rule.ID, model.ChainError); err != nil {
			return err
		}
	}
	return nil
}

func importRuleAccesses(db *database.Session, list []string, ruleID uint64) error {

	for _, src := range list {

		arr := strings.Split(src, "::")
		if len(arr) < 2 {
			return fmt.Errorf("rule auth is not in a valid format")
		}
		var access *model.RuleAccess
		var err error
		switch arr[0] {
		case "remote":
			access, err = createRemoteAccess(db, arr, ruleID)
		case "local":
			access, err = createLocalAccess(db, arr, ruleID)
		default:
			err = fmt.Errorf("rule auth is not in a valid format")
		}
		if err != nil {
			return err
		}
		// If ruleAcess does not exist create
		exists, err := db.Exists(access)
		if err != nil {
			return err
		}
		if !exists {
			err := db.Create(access)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func createRemoteAccess(db *database.Session, arr []string, ruleID uint64) (*model.RuleAccess, error) {
	agent := &model.RemoteAgent{
		Name: arr[1],
	}
	if err := db.Get(agent); err != nil {
		return nil, err
	}
	if len(arr) < 3 {
		// RemoteAgent Access
		return &model.RuleAccess{
			RuleID:     ruleID,
			ObjectType: "remote_agents",
			ObjectID:   agent.ID,
		}, nil
	}
	// RemoteAccount Access
	account := &model.RemoteAccount{
		RemoteAgentID: agent.ID,
		Login:         arr[2],
	}
	if err := db.Get(account); err != nil {
		return nil, err
	}
	return &model.RuleAccess{
		RuleID:     ruleID,
		ObjectType: "remote_accounts",
		ObjectID:   account.ID,
	}, nil
}

func createLocalAccess(db *database.Session, arr []string, ruleID uint64) (*model.RuleAccess, error) {
	agent := &model.LocalAgent{
		Name: arr[1],
	}
	if err := db.Get(agent); err != nil {
		return nil, err
	}
	if len(arr) < 3 {
		// LocalAgent Access
		return &model.RuleAccess{
			RuleID:     ruleID,
			ObjectType: "local_agents",
			ObjectID:   agent.ID,
		}, nil
	}
	// LocalAccount Access
	account := &model.LocalAccount{
		LocalAgentID: agent.ID,
		Login:        arr[2],
	}
	if err := db.Get(account); err != nil {
		return nil, err
	}
	return &model.RuleAccess{
		RuleID:     ruleID,
		ObjectType: "local_accounts",
		ObjectID:   account.ID,
	}, nil
}

func importRuleTasks(db *database.Session, list []ruleTask, ruleID uint64, chain model.Chain) error {

	if len(list) == 0 {
		return nil
	}

	task := &model.Task{
		RuleID: ruleID,
		Chain:  chain,
	}
	if err := db.Delete(task); err != nil {
		if err != database.ErrNotFound {
			return err
		}
	}

	for i, src := range list {

		// Populate
		task.Rank = uint32(i)
		task.Type = src.Type
		task.Args = src.Args

		// Create/Update
		fmt.Printf("Create task type %s at chain %s rank %d\n", task.Type, chain, i)
		if err := db.Create(task); err != nil {
			return err
		}
	}
	return nil
}
