package backup

import (
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func importRules(logger *log.Logger, db database.Access, list []file.Rule) database.Error {

	for _, src := range list {

		//  Create model with basic info to check existence
		var rule model.Rule

		// Check if rule exists
		exists := true
		err := db.Get(&rule, "name=? AND send=?", src.Name, src.IsSend).Run()
		if database.IsNotFound(err) {
			exists = false
		} else if err != nil {
			return err
		}

		// Populate
		rule.Name = src.Name
		rule.IsSend = src.IsSend
		rule.Path = src.Path
		rule.LocalDir = src.LocalDir
		rule.RemoteDir = src.RemoteDir
		rule.LocalTmpDir = src.LocalTmpDir
		if src.InPath != "" {
			logger.Warning("JSON field 'rule.inPath' is deprecated, use 'localDir' & " +
				"'remoteDir' instead")
			if src.IsSend {
				rule.RemoteDir = src.InPath
			} else {
				rule.LocalDir = src.InPath
			}
		}
		if src.OutPath != "" {
			logger.Warning("JSON field 'rule.outPath' is deprecated, use 'localDir' & " +
				"'remoteDir' instead")
			if src.IsSend {
				rule.LocalDir = src.OutPath
			} else {
				rule.RemoteDir = src.OutPath
			}
		}
		if src.WorkPath != "" {
			logger.Warning("JSON field 'rule.workPath' is deprecated, use 'localTmpDir' instead")
			rule.LocalTmpDir = src.WorkPath
		}

		// Create/Update
		if exists {
			logger.Infof("Update rule %s\n", rule.Name)
			err = db.Update(&rule).Run()
		} else {
			logger.Infof("Create rule %s\n", rule.Name)
			err = db.Insert(&rule).Run()
		}
		if err != nil {
			return err
		}
		if err = importRuleAccesses(db, src.Accesses, rule.ID); err != nil {
			return err
		}
		if err = importRuleTasks(logger, db, src.Pre, rule.ID, model.ChainPre); err != nil {
			return err
		}
		if err = importRuleTasks(logger, db, src.Post, rule.ID, model.ChainPost); err != nil {
			return err
		}
		if err = importRuleTasks(logger, db, src.Error, rule.ID, model.ChainError); err != nil {
			return err
		}
	}
	return nil
}

func importRuleAccesses(db database.Access, list []string, ruleID uint64) database.Error {

	for _, src := range list {

		arr := strings.Split(src, "::")
		if len(arr) < 2 {
			return database.NewValidationError("rule auth is not in a valid format")
		}
		var access *model.RuleAccess
		var err database.Error
		switch arr[0] {
		case "remote":
			access, err = createRemoteAccess(db, arr, ruleID)
		case "local":
			access, err = createLocalAccess(db, arr, ruleID)
		default:
			err = database.NewValidationError("rule auth is not in a valid format")
		}
		if err != nil {
			return err
		}
		// If ruleAccess does not exist create
		err = db.Get(access, "rule_id=? AND object_type=? AND object_id=?",
			access.RuleID, access.ObjectType, access.ObjectID).Run()
		if database.IsNotFound(err) {
			if err := db.Insert(access).Run(); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func createRemoteAccess(db database.ReadAccess, arr []string,
	ruleID uint64) (*model.RuleAccess, database.Error) {

	var agent model.RemoteAgent
	if err := db.Get(&agent, "name=?", arr[1]).Run(); err != nil {
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
	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", agent.ID, arr[2]).
		Run(); err != nil {
		return nil, err
	}
	return &model.RuleAccess{
		RuleID:     ruleID,
		ObjectType: "remote_accounts",
		ObjectID:   account.ID,
	}, nil
}

func createLocalAccess(db database.ReadAccess, arr []string,
	ruleID uint64) (*model.RuleAccess, database.Error) {

	var agent model.LocalAgent
	if err := db.Get(&agent, "name=?", arr[1]).Run(); err != nil {
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
	var account model.LocalAccount
	if err := db.Get(&account, "local_agent_id=? AND login=?", agent.ID, arr[2]).
		Run(); err != nil {
		return nil, err
	}
	return &model.RuleAccess{
		RuleID:     ruleID,
		ObjectType: "local_accounts",
		ObjectID:   account.ID,
	}, nil
}

func importRuleTasks(logger *log.Logger, db database.Access, list []file.Task,
	ruleID uint64, chain model.Chain) database.Error {

	if len(list) == 0 {
		return nil
	}

	var task model.Task
	if err := db.DeleteAll(&task).Where("rule_id=? AND chain=?", ruleID, chain).Run(); err != nil {
		return err
	}

	for i, src := range list {

		// Populate
		task.RuleID = ruleID
		task.Chain = chain
		task.Rank = uint32(i)
		task.Type = src.Type
		task.Args = src.Args

		// Create/Update
		logger.Infof("Create task type %s at chain %s rank %d\n", task.Type, chain, i)
		if err := db.Insert(&task).Run(); err != nil {
			return err
		}
	}
	return nil
}
