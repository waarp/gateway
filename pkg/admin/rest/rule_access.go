package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getAuthorizedRules(db *database.DB, objType string, objID uint64) (*api.AuthorizedRules, error) {
	var rules model.Rules
	query := db.Select(&rules).Where("(id IN (SELECT DISTINCT rule_id FROM "+model.TableRuleAccesses+
		" WHERE object_id = ? AND object_type = ?)) OR (SELECT COUNT(*) FROM "+model.TableRuleAccesses+
		" WHERE rule_id = id) = 0", objID, objType)

	if err := query.Run(); err != nil {
		return nil, err
	}

	authorized := &api.AuthorizedRules{}

	for _, rule := range rules {
		if rule.IsSend { // if send == true
			authorized.Sending = append(authorized.Sending, rule.Name)
		} else {
			authorized.Reception = append(authorized.Reception, rule.Name)
		}
	}

	return authorized, nil
}

func getAuthorizedRuleList(db *database.DB, objType string, ids []uint64) ([]api.AuthorizedRules, error) {
	rules := make([]api.AuthorizedRules, len(ids))

	for i, obj := range ids {
		r, err := getAuthorizedRules(db, objType, obj)
		if err != nil {
			return nil, err
		}

		rules[i] = *r
	}

	return rules, nil
}

func authorizeRule(w http.ResponseWriter, r *http.Request, db *database.DB,
	target string, id uint64) error {
	rule, err := getRl(r, db)
	if err != nil {
		return err
	}

	n, err1 := db.Count(&model.RuleAccess{}).Where("rule_id=?", rule.ID).Run()
	if err1 != nil {
		return err1
	}

	access := &model.RuleAccess{
		RuleID:     rule.ID,
		ObjectID:   id,
		ObjectType: target,
	}
	if err := db.Insert(access).Run(); err != nil {
		return err
	}

	if n == 0 {
		fmt.Fprintf(w, "Usage of the %s rule '%s' is now restricted.",
			ruleDirection(rule), rule.Name)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return nil
}

func revokeRule(w http.ResponseWriter, r *http.Request, db *database.DB,
	target string, id uint64) error {
	rule, err := getRl(r, db)
	if err != nil {
		return err
	}

	if err := db.DeleteAll(&model.RuleAccess{}).Where("rule_id=? AND object_id=? "+
		"AND object_type=?", rule.ID, id, target).Run(); err != nil {
		return err
	}

	n, err1 := db.Count(&model.RuleAccess{}).Where("rule_id=?", rule.ID).Run()
	if err1 != nil {
		return err1
	}

	if n == 0 {
		fmt.Fprintf(w, "Usage of the %s rule '%s' is now unrestricted.",
			ruleDirection(rule), rule.Name)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return nil
}

func makeServerAccess(db *database.DB, rule *model.Rule) ([]string, error) {
	var agents model.LocalAgents
	if err := db.Select(&agents).Where("id IN (SELECT object_id FROM "+model.TableRuleAccesses+
		" WHERE rule_id=? AND object_type=?)", rule.ID, model.TableLocAgents).Run(); err != nil {
		return nil, err
	}

	names := make([]string, len(agents))
	for i := range agents {
		names[i] = agents[i].Name
	}

	return names, nil
}

func makePartnerAccess(db *database.DB, rule *model.Rule) ([]string, error) {
	var agents model.RemoteAgents
	if err := db.Select(&agents).Where("id IN (SELECT object_id FROM "+model.TableRuleAccesses+
		" WHERE rule_id=? AND object_type=?)", rule.ID, model.TableRemAgents).Run(); err != nil {
		return nil, err
	}

	names := make([]string, len(agents))
	for i, agent := range agents {
		names[i] = agent.Name
	}

	return names, nil
}

func convertAgentIDs(db *database.DB, isLocal bool, access map[uint64][]string) (map[string][]string, error) {
	if len(access) == 0 {
		return map[string][]string{}, nil
	}

	ids := make([]uint64, len(access))
	i := 0

	for id := range access {
		ids[i] = id
		i++
	}

	names := map[string][]string{}

	if isLocal {
		var agents model.LocalAgents
		if err := db.Select(&agents).In("id", ids).Run(); err != nil {
			return nil, err
		}

		for i := range agents {
			names[agents[i].Name] = access[agents[i].ID]
		}
	} else {
		var agents model.RemoteAgents
		if err := db.Select(&agents).In("id", ids).Run(); err != nil {
			return nil, err
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
	if err := db.Select(&accounts).Where("id IN (SELECT object_id FROM "+model.TableRuleAccesses+
		" WHERE rule_id=? AND object_type=?)", rule.ID, model.TableLocAccounts).Run(); err != nil {
		return nil, err
	}

	accessIDs := map[uint64][]string{}
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
	if err := db.Select(&accounts).Where("id IN (SELECT object_id FROM "+model.TableRuleAccesses+
		" WHERE rule_id=? AND object_type=?)", rule.ID, model.TableRemAccounts).Run(); err != nil {
		return nil, err
	}

	accessIDs := map[uint64][]string{}
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
