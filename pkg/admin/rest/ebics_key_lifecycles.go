package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
)

func getEbicsKeyLifecycle(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lifecycle, err := getDBEbicsKeyLifecycle(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsKeyLifecycleToREST(lifecycle)))
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsKeyLifecycles(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "id", asc: false},
		"id+":     order{col: "id", asc: true},
		"id-":     order{col: "id", asc: false},
		"status+": order{col: "status", asc: true},
		"status-": order{col: "status", asc: false},
		"usage+":  order{col: "key_usage", asc: true},
		"usage-":  order{col: "key_usage", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var lifecycles model.EbicsKeyLifecycles

		query, err := parseSelectQuery(r, db, validSorting, &lifecycles)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsKeyLifecycle, len(lifecycles))
		for i, lifecycle := range lifecycles {
			out[i] = DBEbicsKeyLifecycleToREST(lifecycle)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"keyLifecycles": out}))
	}
}

func actOnEbicsKeyLifecycle(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lifecycle, err := getDBEbicsKeyLifecycle(r, db)
		if handleError(w, logger, err) {
			return
		}

		action := &api.InEbicsKeyLifecycleAction{}
		if err = readJSON(r, action); handleError(w, logger, err) {
			return
		}

		err = ebicsruntime.ApplyKeyLifecycleAction(lifecycle, ebicsruntime.KeyLifecycleAction{
			Action:   action.Action,
			Operator: action.Operator,
			Reason:   action.Reason,
			Evidence: action.Evidence,
		})
		if handleError(w, logger, err) {
			return
		}

		if err = db.Update(lifecycle).Run(); handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsKeyLifecycleToREST(lifecycle)))
	}
}
