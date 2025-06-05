package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getAuthorizedRules(db database.ReadAccess, target model.AccessTarget) (api.AuthorizedRules, error) {
	rules, err := target.GetAuthorizedRules(db)
	if err != nil {
		return api.AuthorizedRules{}, internalf("%v", err)
	}

	authorized := api.AuthorizedRules{}

	for _, rule := range rules {
		if rule.IsSend { // if send == true
			authorized.Sending = append(authorized.Sending, rule.Name)
		} else {
			authorized.Reception = append(authorized.Reception, rule.Name)
		}
	}

	return authorized, nil
}

func authorizeRule(w http.ResponseWriter, r *http.Request, db *database.DB,
	target model.AccessTarget,
) error {
	rule, getErr := retrieveDBRule(r, db)
	if getErr != nil {
		return getErr
	}

	n, countErr := db.Count(&model.RuleAccess{}).Where("rule_id=?", rule.ID).Run()
	if countErr != nil {
		return fmt.Errorf("failed to count rule accesses: %w", countErr)
	}

	access := &model.RuleAccess{RuleID: rule.ID}
	target.SetAccessTarget(access)

	if err := db.Insert(access).Run(); err != nil {
		return fmt.Errorf("failed to insert rule access: %w", err)
	}

	if n == 0 {
		fmt.Fprintf(w, "Usage of the %s rule %q is now restricted.",
			ruleDirection(rule), rule.Name)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return nil
}

func revokeRule(w http.ResponseWriter, r *http.Request, db *database.DB,
	target model.AccessTarget,
) error {
	rule, getErr := retrieveDBRule(r, db)
	if getErr != nil {
		return getErr
	}

	if err := db.DeleteAll(&model.RuleAccess{}).Where("rule_id=?", rule.ID).
		Where(target.GenAccessSelectCond()).Run(); err != nil {
		return fmt.Errorf("failed to delete rule accesses: %w", err)
	}

	n, countErr := db.Count(&model.RuleAccess{}).Where("rule_id=?", rule.ID).Run()
	if countErr != nil {
		return fmt.Errorf("failed to count rule accesses: %w", countErr)
	}

	if n == 0 {
		fmt.Fprintf(w, "Usage of the %s rule %q is now unrestricted.",
			ruleDirection(rule), rule.Name)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return nil
}

func makeServerAccess(db *database.DB, rule *model.Rule) ([]string, error) {
	var agents model.LocalAgents
	if err := db.Select(&agents).Where("id IN (SELECT local_agent_id FROM rule_access"+
		" WHERE rule_id=? AND local_agent_id IS NOT NULL)", rule.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve servers: %w", err)
	}

	names := make([]string, len(agents))
	for i := range agents {
		names[i] = agents[i].Name
	}

	return names, nil
}

func makePartnerAccess(db *database.DB, rule *model.Rule) ([]string, error) {
	var agents model.RemoteAgents
	if err := db.Select(&agents).Where("id IN (SELECT remote_agent_id FROM rule_access"+
		" WHERE rule_id=? AND remote_agent_id IS NOT NULL)", rule.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve partners: %w", err)
	}

	names := make([]string, len(agents))
	for i, agent := range agents {
		names[i] = agent.Name
	}

	return names, nil
}

func convertAgentIDs(db *database.DB, isLocal bool, access map[int64][]string) (map[string][]string, error) {
	if len(access) == 0 {
		return map[string][]string{}, nil
	}

	ids := make([]int64, len(access))
	i := 0

	for id := range access {
		ids[i] = id
		i++
	}

	names := map[string][]string{}

	if isLocal {
		var agents model.LocalAgents
		if err := db.Select(&agents).In("id", ids).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve servers: %w", err)
		}

		for i := range agents {
			names[agents[i].Name] = access[agents[i].ID]
		}
	} else {
		var agents model.RemoteAgents
		if err := db.Select(&agents).In("id", ids).Run(); err != nil {
			return nil, fmt.Errorf("failed to retrieve partners: %w", err)
		}

		for _, agent := range agents {
			names[agent.Name] = access[agent.ID]
		}
	}

	return names, nil
}

//nolint:dupl // duplicated code is about a different type
func makeLocalAccountAccess(db *database.DB, rule *model.Rule) (map[string][]string, error) {
	var accounts model.LocalAccounts
	if err := db.Select(&accounts).Where("id IN (SELECT local_account_id FROM rule_access"+
		" WHERE rule_id=? AND local_account_id IS NOT NULL)", rule.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve local accounts: %w", err)
	}

	accessIDs := map[int64][]string{}
	for _, account := range accounts {
		if _, ok := accessIDs[account.LocalAgentID]; !ok {
			accessIDs[account.LocalAgentID] = []string{account.Login}
		} else {
			accessIDs[account.LocalAgentID] = append(accessIDs[account.LocalAgentID], account.Login)
		}
	}

	return convertAgentIDs(db, true, accessIDs)
}

//nolint:dupl // duplicated code is about a different type
func makeRemoteAccountAccess(db *database.DB, rule *model.Rule) (map[string][]string, error) {
	var accounts model.RemoteAccounts
	if err := db.Select(&accounts).Where("id IN (SELECT remote_account_id FROM rule_access"+
		" WHERE rule_id=? AND remote_account_id IS NOT NULL)", rule.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve remote accounts: %w", err)
	}

	accessIDs := map[int64][]string{}
	for _, account := range accounts {
		if _, ok := accessIDs[account.RemoteAgentID]; !ok {
			accessIDs[account.RemoteAgentID] = []string{account.Login}
		} else {
			accessIDs[account.RemoteAgentID] = append(accessIDs[account.RemoteAgentID], account.Login)
		}
	}

	return convertAgentIDs(db, false, accessIDs)
}

func makeRuleAccess(db *database.DB, rule *model.Rule) (*api.RuleAccess, error) {
	servers, err := makeServerAccess(db, rule)
	if err != nil {
		return nil, err
	}

	partners, err := makePartnerAccess(db, rule)
	if err != nil {
		return nil, err
	}

	locAccounts, err := makeLocalAccountAccess(db, rule)
	if err != nil {
		return nil, err
	}

	remAccounts, err := makeRemoteAccountAccess(db, rule)
	if err != nil {
		return nil, err
	}

	return &api.RuleAccess{
		LocalServers:   servers,
		RemotePartners: partners,
		LocalAccounts:  locAccounts,
		RemoteAccounts: remAccounts,
	}, nil
}
