package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func getTransIDs(db *database.DB, trans *InTransfer) (uint64, uint64, uint64, error) {
	rule := &model.Rule{Name: trans.Rule, IsSend: trans.IsSend}
	if err := db.Get(rule); err != nil {
		if err == database.ErrNotFound {
			return 0, 0, 0, badRequest("no rule '%s' found", rule.Name)
		}
		return 0, 0, 0, err
	}

	partner := &model.RemoteAgent{Name: trans.Partner}
	if err := db.Get(partner); err != nil {
		if err == database.ErrNotFound {
			return 0, 0, 0, badRequest("no partner '%s' found", partner.Name)
		}
		return 0, 0, 0, err
	}
	account := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: trans.Account}
	if err := db.Get(account); err != nil {
		if err == database.ErrNotFound {
			return 0, 0, 0, badRequest("no account '%s' found for partner %s",
				account.Login, partner.Name)
		}
		return 0, 0, 0, err
	}
	return rule.ID, account.ID, partner.ID, nil
}

// getTransNames returns (in this order) the transfer's rule name, requester and
// requested.
func getTransNames(db *database.DB, trans *model.Transfer) (string, string, string, error) {
	rule := &model.Rule{ID: trans.RuleID}
	if err := db.Get(rule); err != nil {
		return "", "", "", err
	}

	if trans.IsServer {
		requester := &model.LocalAccount{ID: trans.AccountID}
		if err := db.Get(requester); err != nil {
			return "", "", "", fmt.Errorf("no loc account %v", trans.AccountID)
		}
		requested := &model.LocalAgent{ID: trans.AgentID}
		if err := db.Get(requested); err != nil {
			return "", "", "", fmt.Errorf("no server %v", trans.AccountID)
		}
		return rule.Name, requester.Login, requested.Name, nil
	}
	requester := &model.RemoteAccount{ID: trans.AccountID}
	if err := db.Get(requester); err != nil {
		return "", "", "", fmt.Errorf("no rem account %v", trans.AccountID)
	}
	requested := &model.RemoteAgent{ID: trans.AgentID}
	if err := db.Get(requested); err != nil {
		return "", "", "", fmt.Errorf("no partner %v", trans.AgentID)
	}
	return rule.Name, requester.Login, requested.Name, nil
}

//nolint:funlen
func parseTransferListQuery(r *http.Request) (filters *database.Filters, err error) {
	filters = &database.Filters{}

	sorting := map[string]string{
		"default": "id ASC",
		"id+":     "id ASC",
		"id-":     "id DESC",
		"status+": "status ASC",
		"status-": "status DESC",
		"start+":  "start ASC",
		"start-":  "start DESC",
	}
	where := builder.NewCond()
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	filters.Limit = 20
	filters.Offset = 0
	if limStr := r.FormValue("limit"); limStr != "" {
		filters.Limit, err = strconv.Atoi(limStr)
		if err != nil {
			return nil, badRequest("'limit' must be an int")
		}
	}
	if offStr := r.FormValue("offset"); offStr != "" {
		filters.Offset, err = strconv.Atoi(offStr)
		if err != nil {
			return nil, badRequest("'offset' must be an int")
		}
	}

	if rules, ok := r.Form["rule"]; ok {
		where = where.And(builder.In("rule_id", builder.Select("id").From("rules").
			Where(builder.In("name", rules))))
	}
	if statuses, ok := r.Form["status"]; ok {
		where = where.And(builder.In("status", statuses))
	}
	if startStr := r.FormValue("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, badRequest("'%s' is not a valid date", startStr)
		}
		where = where.And(builder.Gte{"start": start.UTC()})
	}
	filters.Conditions = where

	filters.Order = sorting["default"]
	if sortStr := r.FormValue("sort"); sortStr != "" {
		sort, ok := sorting[sortStr]
		if !ok {
			return nil, badRequest("'%s' is not a valid order", sortStr)
		}
		filters.Order = sort
	}

	return filters, nil
}
