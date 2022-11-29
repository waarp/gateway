package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func getTransIDs(db *database.DB, trans *api.InTransfer) (int64, int64, error) {
	if trans.IsSend == nil {
		return 0, 0, badRequest("the transfer direction (isSend) is missing")
	}

	var rule model.Rule
	if err := db.Get(&rule, "name=? AND is_send=?", trans.Rule, trans.IsSend).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, badRequest("no rule '%s' found", trans.Rule)
		}

		return 0, 0, err
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", trans.Partner).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, badRequest("no partner '%s' found", trans.Partner)
		}

		return 0, 0, err
	}

	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", partner.ID,
		trans.Account).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, badRequest("no account '%s' found for partner %s",
				trans.Account, trans.Partner)
		}

		return 0, 0, err
	}

	return rule.ID, account.ID, nil
}

// getTransNames returns (in this order) the transfer's rule, and the names of
// the requester, the requested, and the protocol.
func getTransNames(db *database.DB, trans *model.Transfer) (rule *model.Rule,
	requester, requested, protocol string, err error,
) {
	rule = &model.Rule{}

	if err := db.Get(rule, "id=?", trans.RuleID).Run(); err != nil {
		return nil, "", "", "", err
	}

	if trans.IsServer() {
		var acc model.LocalAccount
		if err = db.Get(&acc, "id=?", trans.LocalAccountID).Run(); err != nil {
			return
		}

		var ag model.LocalAgent
		if err = db.Get(&ag, "id=?", acc.LocalAgentID).Run(); err != nil {
			return
		}

		return rule, acc.Login, ag.Name, ag.Protocol, nil
	}

	var acc model.RemoteAccount
	if err = db.Get(&acc, "id=?", trans.RemoteAccountID).Run(); err != nil {
		return
	}

	var ag model.RemoteAgent
	if err = db.Get(&ag, "id=?", acc.RemoteAgentID).Run(); err != nil {
		return
	}

	return rule, acc.Login, ag.Name, ag.Protocol, nil
}

//nolint:funlen // FIXME should be refactored
func parseTransferListQuery(r *http.Request, db *database.DB,
	transfers *model.Transfers,
) (*database.SelectQuery, error) {
	query := db.Select(transfers).Where("owner=?", conf.GlobalConfig.GatewayName)

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
		return nil, fmt.Errorf("cannot parse form data: %w", err)
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

		query.Where("start >= ?", start.UTC())
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

var errServiceNotFound = errors.New("cannot find the service associated with the transfer")

func getPipelineMap(db *database.DB, protoServices map[int64]proto.Service,
	trans *model.Transfer,
) (*service.TransferMap, error) {
	if !trans.IsServer() {
		return pipeline.ClientTransfers, nil
	}

	var agent model.LocalAgent
	if err := db.Get(&agent, "id=(SELECT local_agent_id FROM local_accounts WHERE id=?)",
		trans.LocalAccountID).Run(); err != nil {
		return nil, err
	}

	serv, ok := protoServices[agent.ID]
	if !ok {
		return nil, errServiceNotFound
	}

	return serv.ManageTransfers(), nil
}
