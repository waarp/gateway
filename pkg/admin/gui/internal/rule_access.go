package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func ListAllAccesses(db database.ReadAccess, rule *model.Rule) ([]*model.RuleAccess, error) {
	var accesses model.RuleAccesses

	return accesses, db.Select(&accesses).Where("rule_id=?", rule.ID).Run()
}

func ListAuthorizedServers(db database.ReadAccess, rule *model.Rule) ([]*model.LocalAgent, error) {
	var servers model.LocalAgents

	return servers, db.Select(&servers).
		Where("id IN (SELECT local_agent_id FROM rule_access WHERE rule_id=?)", rule.ID).
		Run()
}

func ListAuthorizedPartners(db database.ReadAccess, rule *model.Rule) ([]*model.RemoteAgent, error) {
	var partners model.RemoteAgents

	return partners, db.Select(&partners).
		Where("id IN (SELECT remote_agent_id FROM rule_access WHERE rule_id=?)", rule.ID).
		Run()
}

func ListAuthorizedLocalAccounts(db database.ReadAccess, rule *model.Rule) ([]*model.LocalAccount, error) {
	var accounts model.LocalAccounts

	return accounts, db.Select(&accounts).
		Where("id IN (SELECT local_account_id FROM rule_access WHERE rule_id=?)", rule.ID).
		Run()
}

func ListAuthorizedRemoteAccounts(db database.ReadAccess, rule *model.Rule) ([]*model.RemoteAccount, error) {
	var accounts model.RemoteAccounts

	return accounts, db.Select(&accounts).
		Where("id IN (SELECT remote_account_id FROM rule_access WHERE rule_id=?)", rule.ID).
		Run()
}

func AddRuleAccess(db database.Access, rule *model.Rule, target model.AccessTarget) error {
	access := &model.RuleAccess{RuleID: rule.ID}
	target.SetAccessTarget(access)

	return db.Insert(access).Run()
}

func DeleteRuleAccess(db database.Access, rule *model.Rule, target model.AccessTarget) error {
	return db.DeleteAll(&model.RuleAccess{}).
		Where("rule_id=?", rule.ID).
		Where(target.GenAccessSelectCond()).
		Run()
}
