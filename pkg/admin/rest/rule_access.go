package rest

import (
	"fmt"
	"net/http"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func getAuthorizedRules(db *database.DB, objType string, objID uint64) (*AuthorizedRules, error) {
	query := "(id IN (SELECT DISTINCT rule_id FROM rule_access WHERE " +
		"object_id = ? AND object_type = ?)) OR (SELECT COUNT(*) FROM " +
		"rule_access WHERE rule_id = id) = 0"
	cond := builder.Expr(query, objID, objType)
	filters := &database.Filters{Conditions: cond}
	var rules []model.Rule

	if err := db.Select(&rules, filters); err != nil {
		return nil, err
	}

	authorized := &AuthorizedRules{}
	for _, rule := range rules {
		if rule.IsSend { // if send == true
			authorized.Sending = append(authorized.Sending, rule.Name)
		} else {
			authorized.Reception = append(authorized.Reception, rule.Name)
		}
	}
	return authorized, nil
}

func getAuthorizedRuleList(db *database.DB, objType string, ids []uint64) ([]AuthorizedRules, error) {
	rules := make([]AuthorizedRules, len(ids))
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

	a, err := db.Query("SELECT rule_id FROM rule_access WHERE rule_id = ?", rule.ID)
	if err != nil {
		return err
	}

	access := &model.RuleAccess{
		RuleID:     rule.ID,
		ObjectID:   id,
		ObjectType: target,
	}
	if err := db.Create(access); err != nil {
		return err
	}
	if len(a) == 0 {
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

	access := &model.RuleAccess{
		RuleID:     rule.ID,
		ObjectID:   id,
		ObjectType: target,
	}
	if err := db.Delete(access); err != nil {
		return err
	}

	a, err := db.Query("SELECT rule_id FROM rule_access WHERE rule_id = ?", rule.ID)
	if err != nil {
		return err
	}
	if len(a) == 0 {
		fmt.Fprintf(w, "Usage of the %s rule '%s' is now unrestricted.",
			ruleDirection(rule), rule.Name)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return nil
}

func makeAccessIDs(db *database.DB, rule *model.Rule, typ string) ([]uint64, error) {
	var accesses []model.RuleAccess
	filters := &database.Filters{Conditions: builder.Eq{
		"rule_id":     rule.ID,
		"object_type": typ,
	}}
	if err := db.Select(&accesses, filters); err != nil {
		return nil, err
	}

	ids := make([]uint64, len(accesses))
	for i, a := range accesses {
		ids[i] = a.ObjectID
	}
	return ids, nil
}

func makeLocalNames(db *database.DB, ids []uint64) ([]string, error) {
	var agents []model.LocalAgent
	filters := &database.Filters{Conditions: builder.In("id", ids)}

	if err := db.Select(&agents, filters); err != nil {
		return nil, err
	}

	names := make([]string, len(agents))
	for i, agent := range agents {
		names[i] = agent.Name
	}
	return names, nil
}

func makeRemoteNames(db *database.DB, ids []uint64) ([]string, error) {
	var agents []model.RemoteAgent
	filters := &database.Filters{Conditions: builder.In("id", ids)}

	if err := db.Select(&agents, filters); err != nil {
		return nil, err
	}

	names := make([]string, len(agents))
	for i, agent := range agents {
		names[i] = agent.Name
	}
	return names, nil
}

func convertAgentIDs(db *database.DB, isLocal bool, access map[uint64][]string) (map[string][]string, error) {
	ids := make([]uint64, len(access))
	i := 0
	for id := range access {
		ids[i] = id
		i++
	}

	names := map[string][]string{}
	if isLocal {
		var agents []model.LocalAgent
		filters := &database.Filters{Conditions: builder.In("id", ids)}
		if err := db.Select(&agents, filters); err != nil {
			return nil, err
		}
		for _, agent := range agents {
			names[agent.Name] = access[agent.ID]
		}
	} else {
		var agents []model.RemoteAgent
		filters := &database.Filters{Conditions: builder.In("id", ids)}
		if err := db.Select(&agents, filters); err != nil {
			return nil, err
		}
		for _, agent := range agents {
			names[agent.Name] = access[agent.ID]
		}
	}
	return names, nil
}

func makeServerAccess(db *database.DB, rule *model.Rule) ([]string, error) {
	ids, err := makeAccessIDs(db, rule, "local_agents")
	if err != nil {
		return nil, err
	}

	return makeLocalNames(db, ids)
}

func makePartnerAccess(db *database.DB, rule *model.Rule) ([]string, error) {
	ids, err := makeAccessIDs(db, rule, "remote_agents")
	if err != nil {
		return nil, err
	}

	return makeRemoteNames(db, ids)
}

func makeLocalAccountAccess(db *database.DB, rule *model.Rule) (map[string][]string, error) {
	ids, err := makeAccessIDs(db, rule, "local_accounts")
	if err != nil {
		return nil, err
	}

	var accounts []model.LocalAccount
	filters := &database.Filters{Conditions: builder.In("id", ids)}
	if err := db.Select(&accounts, filters); err != nil {
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

func makeRemoteAccountAccess(db *database.DB, rule *model.Rule) (map[string][]string, error) {
	ids, err := makeAccessIDs(db, rule, "remote_accounts")
	if err != nil {
		return nil, err
	}

	var accounts []model.RemoteAccount
	filters := &database.Filters{Conditions: builder.In("id", ids)}
	if err := db.Select(&accounts, filters); err != nil {
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

func makeRuleAccess(db *database.DB, rule *model.Rule) (*RuleAccess, error) {
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

	return &RuleAccess{
		LocalServers:   servers,
		RemotePartners: partners,
		LocalAccounts:  locAccounts,
		RemoteAccounts: remAccounts,
	}, nil
}
