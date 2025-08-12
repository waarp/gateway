package backup

import (
	"errors"
	"fmt"
	"strings"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type accessTarget interface {
	model.AccessTarget
	database.Identifier
}

func resetRules(logger *log.Logger, db database.Access) error {
	var rules model.Rules
	if err := db.Select(&rules).Run(); err != nil {
		logger.Errorf("Failed to retrieve the existing rules: %v", err)

		return fmt.Errorf("failed to retrieve the existing rules: %w", err)
	}

	for _, rule := range rules {
		if err := db.Delete(rule).Run(); err != nil {
			logger.Errorf("Failed to delete the existing rules: %v", err)

			return fmt.Errorf("failed to delete rule %q: %w", rule.Name, err)
		}
	}

	return nil
}

func importRules(logger *log.Logger, db database.Access, list []file.Rule,
	reset bool,
) error {
	if reset {
		if err := resetRules(logger, db); err != nil {
			return err
		}
	}

	for i := range list {
		var rule model.Rule

		src := &list[i]
		exists := true

		dbErr := db.Get(&rule, "name=? AND is_send=?", src.Name, src.IsSend).Run()
		if database.IsNotFound(dbErr) {
			exists = false
		} else if dbErr != nil {
			return fmt.Errorf("failed to retrieve rule %q: %w", src.Name, dbErr)
		}

		rule.Name = src.Name
		rule.IsSend = src.IsSend
		rule.Path = src.Path
		rule.LocalDir = src.LocalDir
		rule.RemoteDir = src.RemoteDir
		rule.TmpLocalRcvDir = src.TmpLocalRcvDir

		importRuleCheckDeprecated(logger, src, &rule)

		if exists {
			logger.Infof("Update rule %q", rule.Name)
			dbErr = db.Update(&rule).Run()
		} else {
			logger.Infof("Create rule %q", rule.Name)
			dbErr = db.Insert(&rule).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import rule %q: %w", rule.Name, dbErr)
		}

		if err := importRuleAccesses(db, src.Accesses, &rule); err != nil {
			return err
		}

		if err := importRuleTasks(logger, db, src.Pre, &rule, model.ChainPre); err != nil {
			return err
		}

		if err := importRuleTasks(logger, db, src.Post, &rule, model.ChainPost); err != nil {
			return err
		}

		if err := importRuleTasks(logger, db, src.Error, &rule, model.ChainError); err != nil {
			return err
		}
	}

	return nil
}

func importRuleCheckDeprecated(logger *log.Logger, src *file.Rule, rule *model.Rule) {
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
		logger.Warning("JSON field 'rule.workPath' is deprecated, use 'tmpReceiveDir' instead")

		rule.TmpLocalRcvDir = src.WorkPath
	}
}

var ErrInvalidRuleAuthFormat = errors.New("invalid rule auth format")

func importRuleAccesses(db database.Access, list []string, rule *model.Rule) error {
	for _, src := range list {
		arr := strings.Split(src, "::")
		if len(arr) < 2 { //nolint:mnd // no need for a constant, only used once
			return database.NewValidationError("rule permission is not in a valid format")
		}

		var (
			access *model.RuleAccess
			target accessTarget
			err    error
		)

		switch arr[0] {
		case "remote":
			access, target, err = createRemoteAccess(db, arr, rule.ID)
		case "local":
			access, target, err = createLocalAccess(db, arr, rule.ID)
		default:
			err = ErrInvalidRuleAuthFormat
		}

		if err != nil {
			return err
		}
		// If ruleAccess does not exist create
		if err = db.Get(access, "rule_id=?", access.RuleID).And(
			target.GenAccessSelectCond()).Run(); database.IsNotFound(err) {
			if err2 := db.Insert(access).Run(); err2 != nil {
				return fmt.Errorf("failed to create rule access: %w", err2)
			}
		} else if err != nil {
			return fmt.Errorf("failed to retrieve rule access: %w", err)
		}
	}

	return nil
}

//nolint:dupl // duplicated sections are about two different types.
func createRemoteAccess(db database.ReadAccess, arr []string,
	ruleID int64,
) (*model.RuleAccess, accessTarget, error) {
	var agent model.RemoteAgent
	if err := db.Get(&agent, "name=?", arr[1]).Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve remote agent %q: %w", arr[1], err)
	}

	if len(arr) < 3 { //nolint:mnd // no need for a constant, only used once
		// RemoteAgent Access
		return &model.RuleAccess{
			RuleID:        ruleID,
			RemoteAgentID: utils.NewNullInt64(agent.ID),
		}, &agent, nil
	}

	// RemoteAccount Access
	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", agent.ID, arr[2]).
		Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve remote account %q: %w", arr[2], err)
	}

	return &model.RuleAccess{
		RuleID:          ruleID,
		RemoteAccountID: utils.NewNullInt64(account.ID),
	}, &account, nil
}

//nolint:dupl //duplicated sections are about two different types.
func createLocalAccess(db database.ReadAccess, arr []string,
	ruleID int64,
) (*model.RuleAccess, accessTarget, error) {
	var agent model.LocalAgent
	if err := db.Get(&agent, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
		arr[1]).Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve local agent %q: %w", arr[1], err)
	}

	if len(arr) < 3 { //nolint:mnd // no need for a constant, only used once
		// LocalAgent Access
		return &model.RuleAccess{
			RuleID:       ruleID,
			LocalAgentID: utils.NewNullInt64(agent.ID),
		}, &agent, nil
	}
	// LocalAccount Access
	var account model.LocalAccount
	if err := db.Get(&account, "local_agent_id=? AND login=?", agent.ID, arr[2]).
		Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve local account %q: %w", arr[2], err)
	}

	return &model.RuleAccess{
		RuleID:         ruleID,
		LocalAccountID: utils.NewNullInt64(account.ID),
	}, &account, nil
}

func importRuleTasks(logger *log.Logger, db database.Access, list []file.Task,
	rule *model.Rule, chain model.Chain,
) error {
	if list == nil {
		return nil
	}

	var task model.Task
	if err := db.DeleteAll(&task).Where("rule_id=? AND chain=?", rule.ID, chain).Run(); err != nil {
		return fmt.Errorf("failed to purge %s-tasks of rule %q: %w", chain, rule.Name, err)
	}

	for i, src := range list {
		// Populate
		task.RuleID = rule.ID
		task.Chain = chain
		task.Rank = int8(i)
		task.Type = src.Type
		task.Args = src.Args

		// Create/Update
		logger.Infof("Create task type %s at chain %s rank %d", task.Type, chain, i)

		if err := db.Insert(&task).Run(); err != nil {
			return fmt.Errorf("failed to insert task %q: %w", task.Type, err)
		}
	}

	return nil
}
