package rest

import (
	"context"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsmodule "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics"
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

func prepareEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := &api.InEbicsKeyRotationPrepare{}
		if err := readJSON(r, request); handleError(w, logger, err) {
			return
		}

		result, err := ebicsmodule.PrepareCoordinatedKeyRotation(r.Context(), db, &ebicsmodule.KeyRotationPrepareInput{
			EbicsSubscriberID:              request.EbicsSubscriberID,
			RotationType:                   request.RotationType,
			CoordinationID:                 request.CoordinationID,
			NextAuthenticationCredentialID: request.NextAuthenticationCredentialID,
			NextEncryptionCredentialID:     request.NextEncryptionCredentialID,
			NextSignatureCredentialID:      request.NextSignatureCredentialID,
			SignatureOrderType:             request.SignatureOrderType,
			Operator:                       request.Operator,
			Reason:                         request.Reason,
			Evidence:                       request.Evidence,
		})
		if handleError(w, logger, err) {
			return
		}

		out, err := dbEbicsKeyRotationGroupToREST(result)
		if handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, out))
	}
}

func sendEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, ebicsmodule.SendCoordinatedKeyRotation)
}

func confirmEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, ebicsmodule.ConfirmCoordinatedKeyRotation)
}

func cancelEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, ebicsmodule.CancelCoordinatedKeyRotation)
}

func rejectEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, ebicsmodule.RejectCoordinatedKeyRotation)
}

func revokeEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, ebicsmodule.RevokeCoordinatedKeyRotation)
}

func actOnEbicsKeyRotationGroup(
	logger *log.Logger,
	db *database.DB,
	action func(
		context.Context,
		*database.DB,
		*ebicsmodule.KeyRotationActionInput,
	) (*ebicsmodule.KeyRotationGroupResult, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := &api.InEbicsKeyRotationAction{}
		if err := readJSON(r, request); handleError(w, logger, err) {
			return
		}

		result, err := action(r.Context(), db, &ebicsmodule.KeyRotationActionInput{
			EbicsSubscriberID:  request.EbicsSubscriberID,
			CoordinationID:     request.CoordinationID,
			SignatureOrderType: request.SignatureOrderType,
			SignatureData:      request.SignatureData,
			Operator:           request.Operator,
			Reason:             request.Reason,
			Evidence:           request.Evidence,
		})
		if handleError(w, logger, err) {
			return
		}

		out, err := dbEbicsKeyRotationGroupToREST(result)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

func dbEbicsKeyRotationGroupToREST(
	result *ebicsmodule.KeyRotationGroupResult,
) (*api.OutEbicsKeyRotationGroup, error) {
	if result == nil {
		return nil, database.NewValidationError("the coordinated EBICS key rotation result is missing")
	}

	out := &api.OutEbicsKeyRotationGroup{
		CoordinationID: result.CoordinationID,
		Lifecycles:     make([]*api.OutEbicsKeyLifecycle, 0, len(result.Lifecycles)),
		Operations:     make([]*api.OutEbicsOperation, 0, len(result.Operations)),
	}

	for _, lifecycle := range result.Lifecycles {
		out.Lifecycles = append(out.Lifecycles, DBEbicsKeyLifecycleToREST(lifecycle))
	}
	for _, operation := range result.Operations {
		restOperation, err := DBEbicsOperationToREST(operation)
		if err != nil {
			return nil, err
		}
		out.Operations = append(out.Operations, restOperation)
	}

	return out, nil
}
