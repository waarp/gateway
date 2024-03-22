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
)

func getTransferClient(db *database.DB, trans *api.InTransfer, protocol string,
) (*model.Client, error) {
	if trans.Client != "" {
		var client model.Client

		if err := db.Get(&client, "name=? AND owner=?", trans.Client,
			conf.GlobalConfig.GatewayName).Run(); err != nil {
			if database.IsNotFound(err) {
				return nil, badRequest("no client '%s' found", trans.Client)
			}

			return nil, err
		}

		return &client, nil
	}

	// If the user didn't specify a client, we search for one with the
	// appropriate protocol. If we can't find one, or if they are multiple
	// ones, we return an error
	var clients model.Clients
	if err := db.Select(&clients).Where("protocol=? AND owner=?", protocol,
		conf.GlobalConfig.GatewayName).Run(); err != nil {
		return nil, err
	}

	// No client found with the given protocol.
	if len(clients) == 0 {
		return nil, badRequest("no suitable %s client found", protocol)
	}

	// Multiple clients found with the given protocol.
	if len(clients) > 1 {
		var candidates []string
		for _, client := range clients {
			candidates = append(candidates, fmt.Sprintf("%q", client.Name))
		}

		return nil, badRequest("multiple suitable %s clients found (%s), please specify one",
			protocol, strings.Join(candidates, ", "))
	}

	return clients[0], nil
}

func getTransInfo(db *database.DB, trans *api.InTransfer,
) (*model.Rule, *model.RemoteAccount, *model.Client, error) {
	if trans.IsSend == nil {
		return nil, nil, nil, badRequest("the transfer direction (isSend) is missing")
	}

	if trans.Rule == "" {
		return nil, nil, nil, badRequest("the transfer rule is missing")
	}

	if trans.Partner == "" {
		return nil, nil, nil, badRequest("the transfer partner is missing")
	}

	if trans.Account == "" {
		return nil, nil, nil, badRequest("the transfer account is missing")
	}

	var rule model.Rule
	if err := db.Get(&rule, "name=? AND is_send=?", trans.Rule, trans.IsSend).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil, badRequest("no rule '%s' found", trans.Rule)
		}

		return nil, nil, nil, err
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=? AND owner=?", trans.Partner,
		conf.GlobalConfig.GatewayName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil, badRequest("no partner '%s' found", trans.Partner)
		}

		return nil, nil, nil, err
	}

	var account model.RemoteAccount
	if err := db.Get(&account, "remote_agent_id=? AND login=?", partner.ID,
		trans.Account).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil, badRequest("no account '%s' found for partner %s",
				trans.Account, trans.Partner)
		}

		return nil, nil, nil, err
	}

	client, cliErr := getTransferClient(db, trans, partner.Protocol)
	if cliErr != nil {
		return nil, nil, nil, cliErr
	}

	return &rule, &account, client, nil
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
		query.In("rule", rules)
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

func getPipelineMap(db *database.DB, protoServices map[string]proto.Service,
	trans *model.Transfer,
) (*service.TransferMap, error) {
	if !trans.IsServer() {
		var client model.Client
		if err := db.Get(&client, "id=?", trans.ClientID).Run(); err != nil {
			return nil, err
		}

		cli, ok := protoServices[client.Name]
		if !ok {
			return nil, errServiceNotFound
		}

		return cli.ManageTransfers(), nil
	}

	var agent model.LocalAgent
	if err := db.Get(&agent, "id=(SELECT local_agent_id FROM local_accounts WHERE id=?)",
		trans.LocalAccountID).Run(); err != nil {
		return nil, err
	}

	serv, ok := protoServices[agent.Name]
	if !ok {
		return nil, errServiceNotFound
	}

	return serv.ManageTransfers(), nil
}
