package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsmodule "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics"
)

func getEbicsContractViewItems(db database.ReadAccess, viewID int64) ([]*api.OutEbicsContractViewItem, error) {
	var items model.EbicsContractViewItems
	if err := db.Select(&items).Owner().Where("contract_view_id=?", viewID).OrderBy("id", true).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS contract view items for view %d: %w", viewID, err)
	}

	out := make([]*api.OutEbicsContractViewItem, len(items))
	for i, item := range items {
		out[i] = DBEbicsContractViewItemToREST(item)
	}

	return out, nil
}

func getEbicsContractView(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view, err := getDBEbicsContractView(r, db)
		if handleError(w, logger, err) {
			return
		}

		outView, err := DBEbicsContractViewToREST(db, view)
		if handleError(w, logger, err) {
			return
		}

		items, err := getEbicsContractViewItems(db, view.ID)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, map[string]any{
			"contractView": outView,
			"items":        items,
		}))
	}
}

func listEbicsContractViews(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":    order{col: "fetched_at", asc: false},
		"fetchedAt+": order{col: "fetched_at", asc: true},
		"fetchedAt-": order{col: "fetched_at", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var views model.EbicsContractViews

		query, err := parseSelectQuery(r, db, validSorting, &views)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsContractView, len(views))
		for i, view := range views {
			out[i], err = DBEbicsContractViewToREST(db, view)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"contractViews": out}))
	}
}

func refreshEbicsContractViews(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &api.InEbicsContractRefresh{IncludeHEV: true}
		if err := readJSON(r, req); handleError(w, logger, err) {
			return
		}

		if req.EbicsSubscriberID == 0 {
			handleError(w, logger, badRequestf("the EBICS subscriber identifier is missing"))
			return
		}

		result, err := ebicsmodule.RefreshContractViews(r.Context(), db, req.EbicsSubscriberID, req.IncludeHEV)
		if handleError(w, logger, err) {
			return
		}

		body := struct {
			ProtocolCheckOperation *api.OutEbicsOperation      `json:"protocolCheckOperation,omitempty"`
			Operations             []*api.OutEbicsOperation    `json:"operations"`
			ContractViews          []*api.OutEbicsContractView `json:"contractViews"`
		}{
			Operations:    make([]*api.OutEbicsOperation, 0, len(result.ContractOperations)),
			ContractViews: make([]*api.OutEbicsContractView, 0, len(result.ContractViews)),
		}

		if result.ProtocolCheckOperation != nil {
			outOperation, convErr := DBEbicsOperationToREST(result.ProtocolCheckOperation)
			if handleError(w, logger, convErr) {
				return
			}
			body.ProtocolCheckOperation = outOperation
		}

		for _, view := range result.ContractViews {
			outView, convErr := DBEbicsContractViewToREST(db, view)
			if handleError(w, logger, convErr) {
				return
			}
			body.ContractViews = append(body.ContractViews, outView)
		}

		for _, operation := range result.ContractOperations {
			outOperation, convErr := DBEbicsOperationToREST(operation)
			if handleError(w, logger, convErr) {
				return
			}
			body.Operations = append(body.Operations, outOperation)
		}

		handleError(w, logger, writeJSON(w, body))
	}
}
