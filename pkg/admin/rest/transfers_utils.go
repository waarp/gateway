package rest

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getTransIDs(db *database.DB, trans *api.InTransfer) (uint64, uint64, uint64, error) {
	if trans.IsSend == nil {
		return 0, 0, 0, badRequest("the transfer direction (isSend) is missing")
	}
	var rule model.Rule
	if err := db.Get(&rule, "name=? AND send=?", trans.Rule, trans.IsSend).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, 0, badRequest("no rule '%s' found", trans.Rule)
		}
		return 0, 0, 0, err
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", trans.Partner).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, 0, badRequest("no partner '%s' found", trans.Partner)
		}
		return 0, 0, 0, err
	}

	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", partner.ID,
		trans.Account).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, 0, badRequest("no account '%s' found for partner %s",
				trans.Account, trans.Partner)
		}
		return 0, 0, 0, err
	}
	return rule.ID, account.ID, partner.ID, nil
}

// getTransNames returns (in this order) the transfer's rule, and the names of
// the requester and requested.
func getTransNames(db *database.DB, trans *model.Transfer) (*model.Rule, string, string, error) {
	var rule model.Rule
	if err := db.Get(&rule, "id=?", trans.RuleID).Run(); err != nil {
		return nil, "", "", err
	}

	if trans.IsServer {
		var requester model.LocalAccount
		if err := db.Get(&requester, "id=?", trans.AccountID).Run(); err != nil {
			return nil, "", "", err
		}
		var requested model.LocalAgent
		if err := db.Get(&requested, "id=?", trans.AgentID).Run(); err != nil {
			return nil, "", "", err
		}
		return &rule, requester.Login, requested.Name, nil
	}
	var requester model.RemoteAccount
	if err := db.Get(&requester, "id=?", trans.AccountID).Run(); err != nil {
		return nil, "", "", err
	}
	var requested model.RemoteAgent
	if err := db.Get(&requested, "id=?", trans.AgentID).Run(); err != nil {
		return nil, "", "", err
	}
	return &rule, requester.Login, requested.Name, nil
}

//nolint:funlen
func parseTransferListQuery(r *http.Request, db *database.DB,
	transfers *model.Transfers) (*database.SelectQuery, error) {
	query := db.Select(transfers)

	sorting := orders{
		"default": order{col: "start", asc: true},
		"id+":     order{col: "id", asc: true},
		"id-":     order{col: "id", asc: false},
		"status+": order{col: "status", asc: true},
		"status-": order{col: "status", asc: false},
		"start+":  order{col: "start", asc: true},
		"start-":  order{col: "start", asc: false},
	}

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	var err error
	limit := 20
	offset := 0
	if limStr := r.FormValue("limit"); limStr != "" {
		limit, err = strconv.Atoi(limStr)
		if err != nil {
			return nil, badRequest("'limit' must be an int")
		}
	}
	if offStr := r.FormValue("offset"); offStr != "" {
		offset, err = strconv.Atoi(offStr)
		if err != nil {
			return nil, badRequest("'offset' must be an int")
		}
	}
	query.Limit(limit, offset)

	if rules, ok := r.Form["rule"]; ok {
		args := make([]interface{}, len(rules))
		for i := range rules {
			args[i] = rules[i]
		}
		query.Where("rule_id IN (SELECT id FROM "+model.TableRules+" WHERE name IN (?"+
			strings.Repeat(",?", len(rules)-1)+"))", args...)
	}
	if statuses, ok := r.Form["status"]; ok {
		query.In("status", statuses)
	}
	if startStr := r.FormValue("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339Nano, startStr)
		if err != nil {
			return nil, badRequest("'%s' is not a valid date", startStr)
		}
		query.Where("start >= ?", start.UTC().Truncate(time.Microsecond).
			Format(time.RFC3339Nano))
	}

	sort := sorting["default"]
	if sortStr := r.FormValue("sort"); sortStr != "" {
		var ok bool
		sort, ok = sorting[sortStr]
		if !ok {
			return nil, badRequest("'%s' is not a valid order", sortStr)
		}
	}
	query.OrderBy(sort.col, sort.asc)

	return query, nil
}
