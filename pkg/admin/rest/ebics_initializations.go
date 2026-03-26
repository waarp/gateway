package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
)

func getEbicsInitialization(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflow, err := getDBEbicsInitialization(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsInitializationToREST(workflow)))
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsInitializations(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "id", asc: false},
		"id+":     order{col: "id", asc: true},
		"id-":     order{col: "id", asc: false},
		"status+": order{col: "status", asc: true},
		"status-": order{col: "status", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var workflows model.EbicsInitializationWorkflows

		query, err := parseSelectQuery(r, db, validSorting, &workflows)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsInitializationWorkflow, len(workflows))
		for i, workflow := range workflows {
			out[i] = DBEbicsInitializationToREST(workflow)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"initializations": out}))
	}
}

func actOnEbicsInitialization(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflow, err := getDBEbicsInitialization(r, db)
		if handleError(w, logger, err) {
			return
		}

		action := &api.InEbicsInitializationAction{}
		if err = readJSON(r, action); handleError(w, logger, err) {
			return
		}

		err = ebicsruntime.ApplyInitializationAction(workflow, ebicsruntime.InitializationAction{
			Action:   action.Action,
			Operator: action.Operator,
			Reason:   action.Reason,
			Evidence: action.Evidence,
		})
		if handleError(w, logger, err) {
			return
		}

		if err = db.Update(workflow).Run(); handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsInitializationToREST(workflow)))
	}
}
