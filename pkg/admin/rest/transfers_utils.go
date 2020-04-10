package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/go-xorm/builder"
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

func parseTransferListQuery(db *database.DB, r *http.Request) (*Builder, error) {
	sorting := map[string]func(*Builder){
		"default": func(query *Builder) { query = query.OrderBy("id ASC") },
		"id+":     func(query *Builder) { query = query.OrderBy("id ASC") },
		"id-":     func(query *Builder) { query = query.OrderBy("id DESC") },
		"rule+": func(query *Builder) {
			query = query.InnerJoin("rules", "transfers.rule_id = rules.id").OrderBy("rules.id ASC")
		},
		"rule-": func(query *Builder) {
			query = query.InnerJoin("rules", "transfers.rule_id = rules.id").OrderBy("rules.id DESC")
		},
		"status+": func(query *Builder) { query = query.OrderBy("status ASC") },
		"status-": func(query *Builder) { query = query.OrderBy("status DESC") },
		"start+":  func(query *Builder) { query = query.OrderBy("start ASC") },
		"start-":  func(query *Builder) { query = query.OrderBy("start DESC") },
	}
	query := db.NewQuery().Select("transfers.*").From("transfers")
	where := NewCond()

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
		where = where.And(In("rule_id", Select("id").From("rules").Where(In("name", rules))))
	}
	if statuses, ok := r.Form["status"]; ok {
		where = where.And(In("status", statuses))
	}
	if startStr := r.FormValue("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, badRequest("'%s' is not a valid date", startStr)
		}
		where = where.And(Gte{"start": start.UTC()})
	}
	query = query.Where(where)

	if sortStr := r.FormValue("sort"); sortStr != "" {
		if sort, ok := sorting[sortStr]; ok {
			sort(query)
		} else {
			return nil, badRequest("'%s' is not a valid order", sortStr)
		}

	} else {
		sorting["default"](query)
	}

	return query, nil
}

func execTransferListQuery(db *database.DB, query *Builder) ([]model.Transfer, error) {
	results, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %s", err.Error())
	}

	transfers := make([]model.Transfer, len(results))
	for i, result := range results {
		transfers[i] = model.Transfer{
			ID:           uint64(result["id"].(int64)),
			RuleID:       uint64(result["rule_id"].(int64)),
			IsServer:     result["is_server"].(int64) != 0,
			AgentID:      uint64(result["agent_id"].(int64)),
			AccountID:    uint64(result["account_id"].(int64)),
			TrueFilepath: result["true_filepath"].(string),
			SourceFile:   result["source_file"].(string),
			DestFile:     result["dest_file"].(string),
			Start:        result["start"].(time.Time).Local(),
			Step:         model.TransferStep(result["step"].(string)),
			Status:       model.TransferStatus(result["status"].(string)),
			Owner:        result["owner"].(string),
			Progress:     uint64(result["progression"].(int64)),
			TaskNumber:   uint64(result["task_number"].(int64)),
			Error: model.TransferError{
				Code: func() model.TransferErrorCode {
					code := model.TeUnknown
					_ = code.Scan(result["error_code"])
					return code
				}(),
				Details: result["error_details"].(string),
			},
			ExtInfo: func() []byte {
				if val, ok := result["ext_info"].([]byte); ok {
					return val
				}
				return nil
			}(),
		}
	}
	return transfers, nil
}
