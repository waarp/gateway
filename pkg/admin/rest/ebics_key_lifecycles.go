package rest

import (
	"context"
	"database/sql"
	"fmt"
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
		if err = ebicsmodule.RecordKeyLifecycleHistory(db, lifecycle, action.Action); handleError(w, logger, err) {
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
			ClientID:                       request.ClientID,
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
		if err = recordEbicsKeyRotationHistory(
			db,
			result,
			"PREPARE_ROTATION",
			request.ClientID,
			request.Operator,
			request.Reason,
			request.Evidence,
		); handleError(w, logger, err) {
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
	return actOnEbicsKeyRotationGroup(logger, db, "SEND_ROTATION", ebicsmodule.SendCoordinatedKeyRotation)
}

func confirmEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, "CONFIRM_ROTATION", ebicsmodule.ConfirmCoordinatedKeyRotation)
}

func cancelEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, "CANCEL_ROTATION", ebicsmodule.CancelCoordinatedKeyRotation)
}

func rejectEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, "REJECT_ROTATION", ebicsmodule.RejectCoordinatedKeyRotation)
}

func revokeEbicsKeyRotation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return actOnEbicsKeyRotationGroup(logger, db, "REVOKE_ROTATION", ebicsmodule.RevokeCoordinatedKeyRotation)
}

func actOnEbicsKeyRotationGroup(
	logger *log.Logger,
	db *database.DB,
	actionName string,
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
			ClientID:           request.ClientID,
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
		if err = recordEbicsKeyRotationHistory(
			db,
			result,
			actionName,
			request.ClientID,
			request.Operator,
			request.Reason,
			request.Evidence,
		); handleError(w, logger, err) {
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

func recordEbicsKeyRotationHistory(
	db *database.DB,
	result *ebicsmodule.KeyRotationGroupResult,
	action string,
	clientID int64,
	operator, reason string,
	evidence map[string]any,
) error {
	if result == nil {
		return nil
	}

	for _, lifecycle := range result.Lifecycles {
		if lifecycle == nil {
			continue
		}

		subscriber := &model.EbicsSubscriber{}
		if err := db.Get(subscriber, "id=?", lifecycle.EbicsSubscriberID).Owner().Run(); err != nil {
			return fmt.Errorf(
				"load subscriber for coordinated EBICS key rotation history on lifecycle %d: %w",
				lifecycle.ID,
				err,
			)
		}

		entry := &model.EbicsHistoryEntry{
			HistoryType:       model.EbicsHistoryTypeActionForRuntime(),
			OperationType:     "KEY_ROTATION",
			Action:            action,
			Status:            lifecycle.Status,
			Severity:          model.EbicsOperationSeverityInfoForRuntime(),
			ClientID:          sql.NullInt64{Int64: clientID, Valid: clientID > 0},
			EbicsHostID:       subscriber.EbicsHostID,
			EbicsSubscriberID: lifecycle.EbicsSubscriberID,
			LifecycleID:       sql.NullInt64{Int64: lifecycle.ID, Valid: true},
			CoordinationID:    result.CoordinationID,
			Operator:          operator,
			Reason:            reason,
			EvidenceMap:       evidence,
			MetadataMap: map[string]any{
				"keyUsage":            lifecycle.KeyUsage,
				"rotationType":        lifecycle.RotationType,
				"currentCredentialID": lifecycle.CurrentCredentialID,
				"nextCredentialID":    nullInt64ToRESTAny(lifecycle.NextCredentialID),
				"triggerOperationID":  nullInt64ToRESTAny(lifecycle.TriggerOperationID),
				"lastOperationID":     nullInt64ToRESTAny(lifecycle.LastOperationID),
			},
		}
		if lifecycle.LastOperationID.Valid {
			entry.OperationID = lifecycle.LastOperationID
		}

		if err := db.Insert(entry).Run(); err != nil {
			return fmt.Errorf(
				"insert coordinated EBICS key rotation history for lifecycle %d: %w",
				lifecycle.ID,
				err,
			)
		}
	}

	return nil
}

func nullInt64ToRESTAny(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}

	return value.Int64
}
