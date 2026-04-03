package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getEbicsHistoryEntry(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entry, err := getDBEbicsHistoryEntry(r, db)
		if handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsHistoryEntryToREST(db, entry)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

func listEbicsHistoryEntries(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":        {col: "created_at", asc: false},
		"id+":            {col: "id", asc: true},
		"id-":            {col: "id", asc: false},
		"created+":       {col: "created_at", asc: true},
		"created-":       {col: "created_at", asc: false},
		"status+":        {col: "status", asc: true},
		"status-":        {col: "status", asc: false},
		"operationType+": {col: "operation_type", asc: true},
		"operationType-": {col: "operation_type", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var entries model.EbicsHistoryEntries

		query, err := parseSelectQuery(r, db, validSorting, &entries)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsHistoryEntry, len(entries))
		for i, entry := range entries {
			out[i], err = DBEbicsHistoryEntryToREST(db, entry)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"history": out}))
	}
}
