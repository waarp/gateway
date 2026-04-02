package rest

import (
	"maps"
	"net/http"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func ebicsRTNProviderRESTToDB(in *api.InEbicsRTNProvider) *model.EbicsRTNProvider {
	config := maps.Clone(in.Configuration)
	if config == nil {
		config = map[string]any{}
	}
	if in.ClientID != nil {
		config["clientID"] = *in.ClientID
	} else {
		delete(config, "clientID")
	}

	provider := &model.EbicsRTNProvider{
		Name:              in.Name,
		Transport:         in.Transport,
		EbicsSubscriberID: in.SubscriberID,
		ConfigurationMap:  config,
		AutoPullPolicy:    in.AutoPullPolicy,
	}

	if in.Enabled != nil {
		provider.Enabled = *in.Enabled
	}

	return provider
}

func getEbicsRTNEvent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, err := getDBEbicsRTNEvent(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsRTNEventToREST(event)))
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsRTNEvents(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":     order{col: "received_at", asc: false},
		"receivedAt+": order{col: "received_at", asc: true},
		"receivedAt-": order{col: "received_at", asc: false},
		"status+":     order{col: "status", asc: true},
		"status-":     order{col: "status", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var events model.EbicsRTNEvents

		query, err := parseSelectQuery(r, db, validSorting, &events)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsRTNEvent, len(events))
		for i, event := range events {
			out[i] = DBEbicsRTNEventToREST(event)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"events": out}))
	}
}

func actOnEbicsRTNEvent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, err := getDBEbicsRTNEvent(r, db)
		if handleError(w, logger, err) {
			return
		}

		action := &api.InEbicsRTNEventAction{}
		if err = readJSON(r, action); handleError(w, logger, err) {
			return
		}

		switch action.Action {
		case "RETRY":
			event.Status = "RETRYABLE"
			event.Attempts++
			event.NextRetryAt = time.Now().UTC()
			event.LastError = action.Reason
		case "QUARANTINE":
			event.Status = "QUARANTINED"
			event.NextRetryAt = time.Time{}
			event.LastError = action.Reason
		default:
			handleError(w, logger, badRequestf("%q is not a supported EBICS RTN action", action.Action))
			return
		}

		if action.Metadata != nil {
			if event.PayloadMap == nil {
				event.PayloadMap = map[string]any{}
			}

			event.PayloadMap["lastOperatorAction"] = action.Action
			event.PayloadMap["lastOperatorReason"] = action.Reason
			event.PayloadMap["lastOperatorMetadata"] = action.Metadata
		}

		if err = db.Update(event).Run(); handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsRTNEventToREST(event)))
	}
}

func addEbicsRTNProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := &api.InEbicsRTNProvider{}
		if err := readJSON(r, in); handleError(w, logger, err) {
			return
		}

		if err := validateRTNProviderPayload(in); handleError(w, logger, err) {
			return
		}

		provider := ebicsRTNProviderRESTToDB(in)
		if err := db.Insert(provider).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, provider.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getEbicsRTNProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider, err := getDBEbicsRTNProvider(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsRTNProviderToREST(db, provider)))
	}
}

func listEbicsRTNProviders(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var providers model.EbicsRTNProviders

		query, err := parseSelectQuery(r, db, validSorting, &providers)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsRTNProvider, len(providers))
		for i, provider := range providers {
			out[i] = DBEbicsRTNProviderToREST(db, provider)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"providers": out}))
	}
}

func updateEbicsRTNProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := getDBEbicsRTNProvider(r, db)
		if handleError(w, logger, err) {
			return
		}

		enabled := current.Enabled
		in := &api.InEbicsRTNProvider{
			Name:           current.Name,
			Transport:      current.Transport,
			Enabled:        &enabled,
			SubscriberID:   current.EbicsSubscriberID,
			ClientID:       modelRTNProviderClientID(current),
			Configuration:  current.ConfigurationMap,
			AutoPullPolicy: current.AutoPullPolicy,
		}
		if err = readJSON(r, in); handleError(w, logger, err) {
			return
		}

		if err = validateRTNProviderPayload(in); handleError(w, logger, err) {
			return
		}

		updated := ebicsRTNProviderRESTToDB(in)
		updated.ID = current.ID
		updated.LastConnectionAt = current.LastConnectionAt
		updated.LastError = current.LastError

		if err = db.Update(updated).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, updated.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceEbicsRTNProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := getDBEbicsRTNProvider(r, db)
		if handleError(w, logger, err) {
			return
		}

		in := &api.InEbicsRTNProvider{}
		if err = readJSON(r, in); handleError(w, logger, err) {
			return
		}

		if err = validateRTNProviderPayload(in); handleError(w, logger, err) {
			return
		}

		replacement := ebicsRTNProviderRESTToDB(in)
		replacement.ID = current.ID
		replacement.LastConnectionAt = current.LastConnectionAt
		replacement.LastError = current.LastError

		if err = db.Update(replacement).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, replacement.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteEbicsRTNProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider, err := getDBEbicsRTNProvider(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err = db.Delete(provider).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func ensureRTNProviderHasConfig(in *api.InEbicsRTNProvider) error {
	if in == nil {
		return badRequest("missing EBICS RTN provider payload")
	}

	if in.Configuration == nil {
		return badRequest("the EBICS RTN provider configuration is missing")
	}

	return nil
}

func validateRTNProviderPayload(in *api.InEbicsRTNProvider) error {
	if err := ensureRTNProviderHasConfig(in); err != nil {
		return err
	}

	if in.Name == "" {
		return badRequest("the EBICS RTN provider name is missing")
	}

	if in.Transport == "" {
		return badRequest("the EBICS RTN provider transport is missing")
	}

	if in.SubscriberID == 0 {
		return badRequest("the EBICS RTN provider subscriber ID is missing")
	}
	if in.AutoPullPolicy == "AUTO" || in.AutoPullPolicy == "AUTO_FILTERED" {
		if in.ClientID == nil || *in.ClientID == 0 {
			return badRequest("the EBICS RTN provider client ID is missing")
		}
	}

	return nil
}

func modelRTNProviderClientID(provider *model.EbicsRTNProvider) *int64 {
	if provider == nil {
		return nil
	}

	clientID, ok := readRTNProviderConfigInt64(provider.ConfigurationMap, "clientID")
	if !ok {
		return nil
	}

	return &clientID
}

func readRTNProviderConfigInt64(config map[string]any, key string) (int64, bool) {
	if config == nil {
		return 0, false
	}

	value, ok := config[key]
	if !ok {
		return 0, false
	}

	switch raw := value.(type) {
	case int64:
		return raw, true
	case int:
		return int64(raw), true
	case float64:
		return int64(raw), true
	default:
		return 0, false
	}
}
