package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

type AuthorizedRules struct {
	Sending   []string `json:"sending"`
	Reception []string `json:"reception"`
}

func getAuthorizedRules(db *database.DB, objType string, objID uint64) (*AuthorizedRules, error) {
	access := []model.RuleAccess{}
	accessFilters := &database.Filters{Conditions: builder.Eq{"object_type": objType,
		"object_id": objID}}

	if err := db.Select(&access, accessFilters); err != nil {
		return nil, err
	}

	ruleIDs := make([]uint64, len(access))
	for i := range access {
		ruleIDs[i] = access[i].RuleID
	}

	rules := []model.Rule{}
	ruleFilters := &database.Filters{Conditions: builder.In("id", ruleIDs)}
	if err := db.Select(&rules, ruleFilters); err != nil {
		return nil, err
	}

	authorized := &AuthorizedRules{}
	for _, r := range rules {
		if r.IsSend {
			authorized.Sending = append(authorized.Sending, r.Name)
		} else {
			authorized.Reception = append(authorized.Reception, r.Name)
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

func authorizeRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			target, id, err := getAgentInfo(r, db)
			if err != nil {
				return err
			}

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
				http.Error(w, fmt.Sprintf("Access to rule '%s' is now restricted.",
					rule.Name), http.StatusOK)
			} else {
				w.WriteHeader(http.StatusOK)
			}

			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}

	}
}

func revokeRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			target, id, err := getAgentInfo(r, db)
			if err != nil {
				return err
			}

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
				http.Error(w, fmt.Sprintf("Access to rule '%s' is now unrestricted.",
					rule.Name), http.StatusOK)
			} else {
				w.WriteHeader(http.StatusOK)
			}

			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}

	}
}

type RuleAccess struct {
	LocalServers   []string            `json:"servers,omitempty"`
	RemotePartners []string            `json:"partners,omitempty"`
	LocalAccounts  map[string][]string `json:"localAccounts,omitempty"`
	RemoteAccounts map[string][]string `json:"remoteAccounts,omitempty"`
}

func makeAccessIDs(db *database.DB, rule *model.Rule, typ string) ([]uint64, error) {
	accesses := []model.RuleAccess{}
	filters := &database.Filters{Conditions: builder.Eq{"rule_id": rule.ID,
		"object_type": typ}}
	if err := db.Select(&accesses, filters); err != nil {
		return nil, err
	}

	ids := make([]uint64, len(accesses))
	for i, a := range accesses {
		ids[i] = a.ObjectID
	}
	return ids, nil
}

func makeNames(db *database.DB, typ string, ids []uint64) ([]string, error) {
	agents, err := db.Query(builder.Select().From(typ).Where(builder.In("id", ids)))
	if err != nil {
		return nil, err
	}

	names := make([]string, len(agents))
	for i, agent := range agents {
		names[i] = agent["name"].(string)
	}
	return names, nil
}

func convertAgentIDs(db *database.DB, typ string, access map[uint64][]string) (map[string][]string, error) {
	ids := make([]uint64, len(access))
	i := 0
	for id := range access {
		ids[i] = id
		i++
	}

	agents, err := db.Query(builder.Select().From(typ).Where(builder.In("id", ids)))
	if err != nil {
		return nil, err
	}

	names := map[string][]string{}
	for _, agent := range agents {
		id := agent["id"].(uint64)
		name := agent["name"].(string)
		names[name] = access[id]
	}
	return names, nil
}

func makeServerAccess(db *database.DB, rule *model.Rule, typ string) ([]string, error) {
	ids, err := makeAccessIDs(db, rule, typ)
	if err != nil {
		return nil, err
	}

	return makeNames(db, typ, ids)
}

func makeAccountAccess(db *database.DB, rule *model.Rule, tblName, agentTblName,
	colName string) (map[string][]string, error) {

	ids, err := makeAccessIDs(db, rule, tblName)
	if err != nil {
		return nil, err
	}

	accounts, err := db.Query(builder.Select().From(tblName).Where(builder.In("id", ids)))
	if err != nil {
		return nil, err
	}

	accessIDs := map[uint64][]string{}
	for _, account := range accounts {
		login := account["login"].(string)
		agentID := account[colName].(uint64)
		if _, ok := accessIDs[agentID]; !ok {
			accessIDs[agentID] = []string{login}
		} else {
			accessIDs[agentID] = append(accessIDs[agentID], login)
		}
	}

	return convertAgentIDs(db, agentTblName, accessIDs)
}

func makeRuleAccess(db *database.DB, rule *model.Rule) (*RuleAccess, error) {
	servers, err := makeServerAccess(db, rule, "local_agents")
	if err != nil {
		return nil, err
	}
	partners, err := makeServerAccess(db, rule, "remote_agents")
	if err != nil {
		return nil, err
	}

	locAccounts, err := makeAccountAccess(db, rule, "local_accounts",
		"local_agents", "local_agent_id")
	if err != nil {
		return nil, err
	}
	remAccounts, err := makeAccountAccess(db, rule, "remote_accounts",
		"remote_agents", "remote_agent_id")
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

func makeRulesAccesses(db *database.DB, rules []model.Rule) (map[uint64]RuleAccess, error) {
	accesses := map[uint64]RuleAccess{}
	for _, r := range rules {
		rule := &r
		access, err := makeRuleAccess(db, rule)
		if err != nil {
			return nil, err
		}
		accesses[rule.ID] = *access
	}
	return accesses, nil
}
