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

func makeRuleAccess(rule *model.Rule) *api.RuleAccess {
	accesses := &api.RuleAccess{
		Servers:        make([]string, len(rule.AuthorizedServers)),
		Partners:       make([]string, len(rule.AuthorizedPartners)),
		LocalAccounts:  map[string][]string{},
		RemoteAccounts: map[string][]string{},
	}

	for i, server := range rule.AuthorizedServers {
		accesses.Servers[i] = server.Name
	}

	for i, partner := range rule.AuthorizedPartners {
		accesses.Partners[i] = partner.Name
	}

	for _, account := range rule.AuthorizedLocalAccounts {
		accounts := accesses.LocalAccounts[account.LocalAgent.Name]
		accesses.LocalAccounts[account.LocalAgent.Name] = append(accounts, account.Login)
	}

	for _, account := range rule.AuthorizedRemoteAccounts {
		accounts := accesses.RemoteAccounts[account.RemoteAgent.Name]
		accesses.RemoteAccounts[account.RemoteAgent.Name] = append(accounts, account.Login)
	}

	return accesses
}
