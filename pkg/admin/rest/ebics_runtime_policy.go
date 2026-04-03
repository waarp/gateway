package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func dbEbicsRuntimePolicyToREST(policy *model.EbicsRuntimePolicy) *api.OutEbicsRuntimePolicy {
	if policy == nil {
		return nil
	}

	return &api.OutEbicsRuntimePolicy{
		Enabled:                     policy.Enabled,
		MaintenanceIntervalSeconds:  policy.MaintenanceIntervalSeconds,
		TransactionRetentionSeconds: policy.TransactionRetentionSeconds,
		RTNEventRetentionSeconds:    policy.RTNEventRetentionSeconds,
	}
}

func retrieveDBEbicsRuntimePolicy(db *database.DB) (*model.EbicsRuntimePolicy, error) {
	policy, err := model.EnsureDefaultEbicsRuntimePolicy(db)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the EBICS runtime policy: %w", err)
	}

	return policy, nil
}

func getEbicsRuntimePolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		policy, err := retrieveDBEbicsRuntimePolicy(db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, dbEbicsRuntimePolicyToREST(policy)))
	}
}

func setEbicsRuntimePolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := retrieveDBEbicsRuntimePolicy(db)
		if handleError(w, logger, err) {
			return
		}

		in := api.PatchEbicsRuntimePolicyReqObject{
			Enabled:                     asNullableBool(current.Enabled),
			MaintenanceIntervalSeconds:  asNullable(current.MaintenanceIntervalSeconds),
			TransactionRetentionSeconds: asNullable(current.TransactionRetentionSeconds),
			RTNEventRetentionSeconds:    asNullable(current.RTNEventRetentionSeconds),
		}
		if err = readJSON(r, &in); handleError(w, logger, err) {
			return
		}

		updated := &model.EbicsRuntimePolicy{ID: current.ID, Name: current.Name}
		setIfValid(&updated.Enabled, in.Enabled)
		setIfValid(&updated.MaintenanceIntervalSeconds, in.MaintenanceIntervalSeconds)
		setIfValid(&updated.TransactionRetentionSeconds, in.TransactionRetentionSeconds)
		setIfValid(&updated.RTNEventRetentionSeconds, in.RTNEventRetentionSeconds)

		if err = db.Update(updated).Run(); handleError(w, logger, err) {
			return
		}

		refreshed, err := retrieveDBEbicsRuntimePolicy(db)
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", r.URL.String())
		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, dbEbicsRuntimePolicyToREST(refreshed)))
	}
}
