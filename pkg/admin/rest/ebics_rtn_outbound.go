package rest

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics"
)

func ebicsRTNOutboundProviderRESTToDB(in *api.InEbicsRTNOutboundProvider) *model.EbicsRTNOutboundProvider {
	config := in.Configuration
	if config == nil {
		config = map[string]any{}
	}

	provider := &model.EbicsRTNOutboundProvider{
		Name:              in.Name,
		Transport:         in.Transport,
		EbicsSubscriberID: in.SubscriberID,
		ConfigurationMap:  config,
	}
	if in.Enabled != nil {
		provider.Enabled = *in.Enabled
	}

	return provider
}

func listEbicsRTNOutboundProviders(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var providers model.EbicsRTNOutboundProviders
		query, err := parseSelectQuery(r, db, validSorting, &providers)
		if handleError(w, logger, err) {
			return
		}
		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsRTNOutboundProvider, len(providers))
		for i, provider := range providers {
			out[i] = DBEbicsRTNOutboundProviderToREST(provider)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"providers": out}))
	}
}

func addEbicsRTNOutboundProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := &api.InEbicsRTNOutboundProvider{}
		if err := readJSON(r, in); handleError(w, logger, err) {
			return
		}
		if err := validateOutboundRTNProviderPayload(in); handleError(w, logger, err) {
			return
		}

		provider := ebicsRTNOutboundProviderRESTToDB(in)
		if err := db.Insert(provider).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, provider.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getEbicsRTNOutboundProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider, err := getDBEbicsRTNOutboundProvider(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsRTNOutboundProviderToREST(provider)))
	}
}

func updateEbicsRTNOutboundProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := getDBEbicsRTNOutboundProvider(r, db)
		if handleError(w, logger, err) {
			return
		}

		enabled := current.Enabled
		in := &api.InEbicsRTNOutboundProvider{
			Name:          current.Name,
			Transport:     current.Transport,
			Enabled:       &enabled,
			SubscriberID:  current.EbicsSubscriberID,
			Configuration: current.ConfigurationMap,
		}
		if err = readJSON(r, in); handleError(w, logger, err) {
			return
		}
		if err = validateOutboundRTNProviderPayload(in); handleError(w, logger, err) {
			return
		}

		updated := ebicsRTNOutboundProviderRESTToDB(in)
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

func replaceEbicsRTNOutboundProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return updateEbicsRTNOutboundProvider(logger, db)
}

func deleteEbicsRTNOutboundProvider(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider, err := getDBEbicsRTNOutboundProvider(r, db)
		if handleError(w, logger, err) {
			return
		}
		if err = db.Delete(provider).Run(); handleError(w, logger, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsRTNOutboundNotifications(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":    order{col: "created_at", asc: false},
		"createdAt+": order{col: "created_at", asc: true},
		"createdAt-": order{col: "created_at", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var notifications model.EbicsRTNOutboundNotifications
		query, err := parseSelectQuery(r, db, validSorting, &notifications)
		if handleError(w, logger, err) {
			return
		}
		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsRTNOutboundNotification, len(notifications))
		for i, notification := range notifications {
			out[i] = DBEbicsRTNOutboundNotificationToREST(notification)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"notifications": out}))
	}
}

func addEbicsRTNOutboundNotification(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := &api.InEbicsRTNOutboundNotification{}
		if err := readJSON(r, in); handleError(w, logger, err) {
			return
		}
		if err := validateOutboundRTNNotificationPayload(in); handleError(w, logger, err) {
			return
		}

		set := &model.EbicsServerReportingSet{}
		if err := db.Get(set, "id=?", in.ServerReportingSetID).Owner().Run(); handleError(w, logger, err) {
			return
		}
		item := &model.EbicsServerReportingItem{}
		if err := db.Get(
			item,
			"server_reporting_set_id=? AND item_key=?",
			set.ID,
			strings.TrimSpace(in.ItemKey),
		).Owner().Run(); handleError(w, logger, err) {
			return
		}

		notification, err := ebics.QueueRTNOutboundNotification(db, in.ProviderID, set, item)
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, strconv.FormatInt(notification.ID, 10)))
		w.WriteHeader(http.StatusCreated)
	}
}

func getEbicsRTNOutboundNotification(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notification, err := getDBEbicsRTNOutboundNotification(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsRTNOutboundNotificationToREST(notification)))
	}
}

func actOnEbicsRTNOutboundNotification(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notification, err := getDBEbicsRTNOutboundNotification(r, db)
		if handleError(w, logger, err) {
			return
		}
		action := &api.InEbicsRTNOutboundNotificationAction{}
		if err = readJSON(r, action); handleError(w, logger, err) {
			return
		}

		switch strings.ToUpper(strings.TrimSpace(action.Action)) {
		case "RETRY":
			notification.Status = model.EbicsRTNOutboundNotificationStatusRetryableForRuntime()
			notification.NextRetryAt = time.Now().UTC()
			notification.LastError = strings.TrimSpace(action.Reason)
		case "QUARANTINE":
			notification.Status = model.EbicsRTNOutboundNotificationStatusQuarantinedForRuntime()
			notification.NextRetryAt = time.Time{}
			notification.LastError = strings.TrimSpace(action.Reason)
		default:
			handleError(w, logger, badRequestf("%q is not a supported outbound RTN action", action.Action))
			return
		}

		if err = db.Update(notification).Run(); handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsRTNOutboundNotificationToREST(notification)))
	}
}

func validateOutboundRTNProviderPayload(in *api.InEbicsRTNOutboundProvider) error {
	if in == nil {
		return badRequest("missing outbound RTN provider payload")
	}
	if in.Name == "" {
		return badRequest("the outbound RTN provider name is missing")
	}
	if in.Transport == "" {
		return badRequest("the outbound RTN provider transport is missing")
	}
	if in.SubscriberID == 0 {
		return badRequest("the outbound RTN provider subscriber ID is missing")
	}
	if in.Configuration == nil {
		return badRequest("the outbound RTN provider configuration is missing")
	}

	return nil
}

func validateOutboundRTNNotificationPayload(in *api.InEbicsRTNOutboundNotification) error {
	if in == nil {
		return badRequest("missing outbound RTN notification payload")
	}
	if in.ProviderID == 0 {
		return badRequest("the outbound RTN notification provider ID is missing")
	}
	if in.ServerReportingSetID == 0 {
		return badRequest("the outbound RTN notification reporting set ID is missing")
	}
	if strings.TrimSpace(in.ItemKey) == "" {
		return badRequest("the outbound RTN notification item key is missing")
	}

	return nil
}
