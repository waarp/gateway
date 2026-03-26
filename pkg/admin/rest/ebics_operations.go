package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getEbicsOperation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operation, err := getDBEbicsOperation(r, db)
		if handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsOperationToREST(operation)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

func listEbicsOperations(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":    order{col: "id", asc: false},
		"id+":        order{col: "id", asc: true},
		"id-":        order{col: "id", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
		"created+":   order{col: "created_at", asc: true},
		"created-":   order{col: "created_at", asc: false},
		"orderType+": order{col: "order_type", asc: true},
		"orderType-": order{col: "order_type", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var operations model.EbicsOperations

		query, err := parseSelectQuery(r, db, validSorting, &operations)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsOperation, len(operations))
		for i, operation := range operations {
			out[i], err = DBEbicsOperationToREST(operation)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"operations": out}))
	}
}
