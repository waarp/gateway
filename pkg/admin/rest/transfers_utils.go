package rest

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/compatibility"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:funlen //no easy way to split the function
func getTransInfo(db *database.DB, trans *api.InTransfer,
) (ruleID int64, accountID, clientID sql.NullInt64, _ error) {
	var null sql.NullInt64

	if !trans.IsSend.Valid {
		return 0, null, null, badRequest("the transfer direction (isSend) is missing")
	}

	if trans.Rule == "" {
		return 0, null, null, badRequest("the transfer rule is missing")
	}

	if trans.Partner == "" {
		return 0, null, null, badRequest("the transfer partner is missing")
	}

	if trans.Account == "" {
		return 0, null, null, badRequest("the transfer account is missing")
	}

	var rule model.Rule
	if err := db.Get(&rule, "name=? AND is_send=?", trans.Rule, trans.IsSend.Value).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, null, null, badRequestf("no rule %q found", trans.Rule)
		}

		return 0, null, null, fmt.Errorf("failed to retrieve rule %q: %w", trans.Rule, err)
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=? AND owner=?", trans.Partner,
		conf.GlobalConfig.GatewayName).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, null, null, badRequestf("no partner %q found", trans.Partner)
		}

		return 0, null, null, fmt.Errorf("failed to retrieve partner %q: %w", trans.Partner, err)
	}

	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", partner.ID,
		trans.Account).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, null, null, badRequestf("no account %q found for partner %q",
				trans.Account, trans.Partner)
		}

		return 0, null, null, fmt.Errorf("failed to retrieve remote account %q: %w",
			trans.Account, err)
	}

	if trans.Client != "" {
		var client model.Client
		if err := db.Get(&client, "name=?", trans.Client).Run(); err != nil {
			return 0, null, null, fmt.Errorf("failed to retrieve client %q: %w", trans.Client, err)
		}

		clientID = utils.NewNullInt64(client.ID)
	} else {
		client, err := compatibility.GetDefaultTransferClient(db, partner.Protocol)
		if err != nil {
			return 0, null, null, fmt.Errorf("failed to retrieve default client: %w", err)
		}

		clientID = utils.NewNullInt64(client.ID)
	}

	return rule.ID, utils.NewNullInt64(account.ID), clientID, nil
}

//nolint:funlen // FIXME should be refactored
func parseTransferListQuery(r *http.Request, db *database.DB,
	transfers *model.NormalizedTransfers,
) (*database.SelectQuery, error) {
	query := db.Select(transfers).Where("owner=?", conf.GlobalConfig.GatewayName)

	//nolint:dupl //kept separate for backwards compatibility
	sorting := orders{
		"default":    order{col: "start", asc: true},
		"id+":        order{col: "id", asc: true},
		"id-":        order{col: "id", asc: false},
		"requested+": order{col: "agent", asc: true},
		"requested-": order{col: "agent", asc: false},
		"requester+": order{col: "account", asc: true},
		"requester-": order{col: "account", asc: false},
		"rule+":      order{col: "rule", asc: true},
		"rule-":      order{col: "rule", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
		"start+":     order{col: "start", asc: true},
		"start-":     order{col: "start", asc: false},
	}

	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("cannot parse form data: %w", err)
	}

	var convErr error

	limit := 20
	offset := 0

	if limStr := r.FormValue("limit"); limStr != "" {
		limit, convErr = strconv.Atoi(limStr)
		if convErr != nil {
			return nil, badRequest("'limit' must be an int")
		}
	}

	if offStr := r.FormValue("offset"); offStr != "" {
		offset, convErr = strconv.Atoi(offStr)
		if convErr != nil {
			return nil, badRequest("'offset' must be an int")
		}
	}

	query.Limit(limit, offset)

	if rules, ok := r.Form["rule"]; ok {
		query.In("rule", rules)
	}

	if statuses, ok := r.Form["status"]; ok {
		query.In("status", statuses)
	}

	if followID := r.FormValue("followID"); followID != "" {
		query.Where("id IN (SELECT CONCAT(transfer_id, history_id) "+
			"FROM transfer_info WHERE name=? AND value=?)",
			model.FollowID, followID)
	}

	if startStr := r.FormValue("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339Nano, startStr)
		if err != nil {
			return nil, badRequestf("%q is not a valid date", startStr)
		}

		query.Where("start >= ?", start.UTC())
	}

	sort := sorting["default"]

	if sortStr := r.FormValue("sort"); sortStr != "" {
		var ok bool
		sort, ok = sorting[sortStr]

		if !ok {
			return nil, badRequestf("%q is not a valid order", sortStr)
		}
	}

	query.OrderBy(sort.col, sort.asc)

	return query, nil
}
