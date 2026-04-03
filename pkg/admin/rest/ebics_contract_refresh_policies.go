package rest

import (
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func contractRefreshPolicyActivation(
	db database.ReadAccess,
	policy *model.EbicsContractRefreshPolicy,
) (status, reason string) {
	if policy == nil {
		return "", ""
	}
	if !policy.Enabled {
		return ebicsActivationDisabled, ""
	}

	var client model.Client
	if err := db.Get(&client, "id=?", policy.ClientID).Run(); err != nil {
		return ebicsRTNProviderActivationBlocked, fmt.Sprintf(
			"the contract refresh policy client %s is missing", strconv.FormatInt(policy.ClientID, 10))
	}
	if client.Disabled {
		return ebicsRTNProviderActivationBlocked, fmt.Sprintf(
			"the contract refresh policy client %s is disabled", strconv.FormatInt(policy.ClientID, 10))
	}

	var subscriber model.EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", policy.EbicsSubscriberID).Run(); err != nil {
		return ebicsRTNProviderActivationBlocked, fmt.Sprintf(
			"the contract refresh policy subscriber %s is missing",
			strconv.FormatInt(policy.EbicsSubscriberID, 10))
	}
	if !subscriber.Enabled {
		return ebicsRTNProviderActivationBlocked, fmt.Sprintf(
			"the contract refresh policy subscriber %s is disabled",
			strconv.FormatInt(policy.EbicsSubscriberID, 10))
	}

	if policy.Status == "ERROR" {
		return ebicsRTNProviderActivationError, policy.LastError
	}

	return ebicsActivationReady, ""
}

func dbEbicsContractRefreshPolicyToREST(
	db database.ReadAccess,
	policy *model.EbicsContractRefreshPolicy,
) *api.OutEbicsContractRefreshPolicy {
	if policy == nil {
		return nil
	}

	out := &api.OutEbicsContractRefreshPolicy{
		ID:              policy.ID,
		Name:            policy.Name,
		Enabled:         policy.Enabled,
		ClientID:        policy.ClientID,
		SubscriberID:    policy.EbicsSubscriberID,
		IncludeHEV:      policy.IncludeHEV,
		IntervalSeconds: policy.IntervalSeconds,
		Status:          policy.Status,
		LastError:       policy.LastError,
	}
	if !policy.NextRunAt.IsZero() {
		next := policy.NextRunAt
		out.NextRunAt = &next
	}
	if !policy.LastAttemptAt.IsZero() {
		lastAttempt := policy.LastAttemptAt
		out.LastAttemptAt = &lastAttempt
	}
	if !policy.LastSuccessAt.IsZero() {
		lastSuccess := policy.LastSuccessAt
		out.LastSuccessAt = &lastSuccess
	}

	var client model.Client
	if err := db.Get(&client, "id=?", policy.ClientID).Run(); err == nil {
		out.ClientName = client.Name
	}

	var subscriber model.EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", policy.EbicsSubscriberID).Run(); err == nil {
		out.PartnerID = subscriber.PartnerID
		out.UserID = subscriber.UserID

		var host model.EbicsHost
		if hostErr := db.Get(&host, "id=?", subscriber.EbicsHostID).Run(); hostErr == nil {
			out.HostID = host.HostID
		}
	}

	out.ActivationStatus, out.ActivationReason = contractRefreshPolicyActivation(db, policy)

	return out
}

func getDBEbicsContractRefreshPolicy(r *http.Request, db *database.DB) (*model.EbicsContractRefreshPolicy, error) {
	name := mux.Vars(r)["ebics_contract_refresh_policy"]
	if name == "" {
		return nil, badRequestf("the EBICS contract refresh policy name is missing")
	}

	policy := &model.EbicsContractRefreshPolicy{}
	if err := db.Get(policy, "name=?", name).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("the EBICS contract refresh policy %q does not exist", name)
		}

		return nil, fmt.Errorf("failed to retrieve the EBICS contract refresh policy %q: %w", name, err)
	}

	return policy, nil
}

func listEbicsContractRefreshPolicies(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var policies model.EbicsContractRefreshPolicies

		query, err := parseSelectQuery(r, db, validSorting, &policies)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsContractRefreshPolicy, 0, len(policies))
		for _, policy := range policies {
			out = append(out, dbEbicsContractRefreshPolicyToREST(db, policy))
		}

		handleError(w, logger, writeJSON(w, map[string]any{"contractRefreshPolicies": out}))
	}
}

func getEbicsContractRefreshPolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy, err := getDBEbicsContractRefreshPolicy(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, dbEbicsContractRefreshPolicyToREST(db, policy)))
	}
}

func addEbicsContractRefreshPolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := &api.InEbicsContractRefreshPolicy{}
		if err := readJSON(r, in); handleError(w, logger, err) {
			return
		}

		policy := &model.EbicsContractRefreshPolicy{
			Name:              in.Name,
			Enabled:           in.Enabled == nil || *in.Enabled,
			ClientID:          in.ClientID,
			EbicsSubscriberID: in.SubscriberID,
			IncludeHEV:        in.IncludeHEV == nil || *in.IncludeHEV,
			IntervalSeconds:   in.IntervalSeconds,
		}
		if err := db.Insert(policy).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", path.Join(r.URL.Path, policy.Name))
		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, dbEbicsContractRefreshPolicyToREST(db, policy)))
	}
}

func updateEbicsContractRefreshPolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := getDBEbicsContractRefreshPolicy(r, db)
		if handleError(w, logger, err) {
			return
		}

		in := api.PatchEbicsContractRefreshPolicyReqObject{
			Name:            asNullable(current.Name),
			Enabled:         asNullableBool(current.Enabled),
			ClientID:        asNullable(current.ClientID),
			SubscriberID:    asNullable(current.EbicsSubscriberID),
			IncludeHEV:      asNullableBool(current.IncludeHEV),
			IntervalSeconds: asNullable(current.IntervalSeconds),
		}
		if err = readJSON(r, &in); handleError(w, logger, err) {
			return
		}

		updated := &model.EbicsContractRefreshPolicy{}
		*updated = *current
		setIfValid(&updated.Name, in.Name)
		setIfValid(&updated.Enabled, in.Enabled)
		setIfValid(&updated.ClientID, in.ClientID)
		setIfValid(&updated.EbicsSubscriberID, in.SubscriberID)
		setIfValid(&updated.IncludeHEV, in.IncludeHEV)
		setIfValid(&updated.IntervalSeconds, in.IntervalSeconds)

		if !updated.Enabled {
			updated.Status = "DISABLED"
		} else if updated.Status == "DISABLED" {
			updated.Status = "READY"
		}

		if err = db.Update(updated).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", path.Join("/api/ebics/contract-refresh-policies", updated.Name))
		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, dbEbicsContractRefreshPolicyToREST(db, updated)))
	}
}

func replaceEbicsContractRefreshPolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return updateEbicsContractRefreshPolicy(logger, db)
}

func deleteEbicsContractRefreshPolicy(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy, err := getDBEbicsContractRefreshPolicy(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, db.Delete(policy).Run())
	}
}
