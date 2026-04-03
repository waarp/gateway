package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getEbicsServerContractSetItems(
	db database.ReadAccess,
	setID int64,
) ([]*api.OutEbicsServerContractItem, error) {
	var items model.EbicsServerContractItems
	if err := db.Select(&items).Owner().Where("server_contract_set_id=?", setID).OrderBy("id", true).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS server contract items for set %d: %w", setID, err)
	}

	out := make([]*api.OutEbicsServerContractItem, len(items))
	for i, item := range items {
		out[i] = DBEbicsServerContractItemToREST(item)
	}

	return out, nil
}

func getEbicsServerContractSet(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		set, err := getDBEbicsServerContractSet(r, db)
		if handleError(w, logger, err) {
			return
		}

		outSet, err := DBEbicsServerContractSetToREST(db, set)
		if handleError(w, logger, err) {
			return
		}

		items, err := getEbicsServerContractSetItems(db, set.ID)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, map[string]any{
			"serverContractSet": outSet,
			"items":             items,
		}))
	}
}

func listEbicsServerContractSets(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":    order{col: "published_at", asc: false},
		"published+": order{col: "published_at", asc: true},
		"published-": order{col: "published_at", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
		"name+":      order{col: "name", asc: true},
		"name-":      order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var sets model.EbicsServerContractSets

		query, err := parseSelectQuery(r, db, validSorting, &sets)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsServerContractSet, len(sets))
		for i, set := range sets {
			out[i], err = DBEbicsServerContractSetToREST(db, set)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"serverContractSets": out}))
	}
}
