package rest

import (
	"context"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func ClientDBToREST(client *model.Client) *api.OutClient {
	return &api.OutClient{
		Name:         client.Name,
		Enabled:      !client.Disabled,
		Protocol:     client.Protocol,
		LocalAddress: client.LocalAddress.String(),
		ProtoConfig:  client.ProtoConfig,
	}
}

func ClientsDBToREST(dbClients model.Clients) []*api.OutClient {
	var restClients []*api.OutClient

	for _, dbClient := range dbClients {
		restClient := ClientDBToREST(dbClient)

		restClients = append(restClients, restClient)
	}

	return restClients
}

func ClientRESTToDB(client *api.InClient) (*model.Client, error) {
	cli := &model.Client{
		Name:        client.Name.Value,
		Protocol:    client.Protocol.Value,
		Disabled:    client.Disabled,
		ProtoConfig: client.ProtoConfig,
	}

	if client.LocalAddress.Valid {
		if err := cli.LocalAddress.Set(client.LocalAddress.Value); err != nil {
			return nil, fmt.Errorf("failed to parse local address: %w", err)
		}
	}

	return cli, nil
}

func getDBClient(r *http.Request, db *database.DB) (*model.Client, error) {
	clientName, ok := mux.Vars(r)["client"]
	if !ok {
		return nil, notFound("missing client name")
	}

	var client model.Client
	if err := db.Get(&client, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
		clientName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("client %q not found", clientName)
		}

		return nil, fmt.Errorf("failed to retrieve client %q: %w", clientName, err)
	}

	return &client, nil
}

func getClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restClient := ClientDBToREST(client)

		handleError(w, logger, writeJSON(w, restClient))
	}
}

func listClients(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"proto+":  order{"protocol", true},
		"proto-":  order{"protocol", false},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbClients model.Clients
		query, queryErr := parseSelectQuery(r, db, validSorting, &dbClients)

		if handleError(w, logger, queryErr) {
			return
		}

		if err := parseProtoParam(r, query); handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restClients := ClientsDBToREST(dbClients)

		resp := map[string][]*api.OutClient{"clients": restClients}
		handleError(w, logger, writeJSON(w, resp))
	}
}

//nolint:dupl //duplicate is for another type
func createClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restClient api.InClient
		if err := readJSON(r, &restClient); handleError(w, logger, err) {
			return
		}

		dbClient, convErr := ClientRESTToDB(&restClient)
		if handleError(w, logger, convErr) {
			return
		}

		service, mkErr := makeClientService(db, dbClient)
		if handleError(w, logger, mkErr) {
			return
		}

		if err := db.Insert(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		services.Clients[dbClient.Name] = service

		w.Header().Set("Location", location(r.URL, dbClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for servers, best keep separate
func deleteClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, service, getErr := getClientService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		switch code, _ := service.State(); code {
		case utils.StateError, utils.StateOffline:
		default:
			ctx, cancel := context.WithTimeout(r.Context(), serviceShutdownTimeout)
			defer cancel()

			if err := service.Stop(ctx); handleError(w, logger, err) {
				return
			}
		}

		if err := db.Delete(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		delete(services.Clients, dbClient.Name)

		w.WriteHeader(http.StatusNoContent)
	}
}

func updateClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDBClient, oldService, getErr := getClientService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restClient := &api.InClient{
			Name:         asNullableStr(oldDBClient.Name),
			Protocol:     asNullableStr(oldDBClient.Protocol),
			LocalAddress: asNullableStr(oldDBClient.LocalAddress.String()),
			ProtoConfig:  oldDBClient.ProtoConfig,
		}
		if err := readJSON(r, restClient); handleError(w, logger, err) {
			return
		}

		dbClient := &model.Client{
			ID:          oldDBClient.ID,
			Name:        restClient.Name.Value,
			Protocol:    restClient.Protocol.Value,
			ProtoConfig: restClient.ProtoConfig,
			Disabled:    oldDBClient.Disabled,
		}

		if restClient.LocalAddress.Valid {
			if err := dbClient.LocalAddress.Set(restClient.LocalAddress.
				Value); handleError(w, logger, err) {
				return
			}
		}

		newService, mkErr := makeClientService(db, dbClient)
		if handleError(w, logger, mkErr) {
			return
		}

		if err := db.Update(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		delete(services.Clients, oldDBClient.Name)
		services.Clients[dbClient.Name] = newService

		if state, _ := oldService.State(); state == utils.StateRunning {
			ctx, cancel := context.WithTimeout(r.Context(), serviceShutdownTimeout)
			defer cancel()

			if err := oldService.Stop(ctx); handleError(w, logger, err) {
				return
			}

			if err := newService.Start(); handleError(w, logger, err) {
				return
			}
		}

		w.Header().Set("Location", location(r.URL, dbClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for a completely different type
func replaceClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDBClient, oldService, getErr := getClientService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restClient := api.InClient{}
		if err := readJSON(r, &restClient); handleError(w, logger, err) {
			return
		}

		dbClient, convErr := ClientRESTToDB(&restClient)
		if handleError(w, logger, convErr) {
			return
		}

		dbClient.ID = oldDBClient.ID

		newService, mkErr := makeClientService(db, dbClient)
		if handleError(w, logger, mkErr) {
			return
		}

		if err := db.Update(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		delete(services.Clients, oldDBClient.Name)
		services.Clients[dbClient.Name] = newService

		if state, _ := oldService.State(); state == utils.StateRunning {
			ctx, cancel := context.WithTimeout(r.Context(), serviceShutdownTimeout)
			defer cancel()

			if err := oldService.Stop(ctx); handleError(w, logger, err) {
				return
			}

			if err := newService.Start(); handleError(w, logger, err) {
				return
			}
		}

		w.Header().Set("Location", location(r.URL, dbClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getClientService(r *http.Request, db *database.DB) (*model.Client, protocol.Client, error) {
	dbClient, getErr := getDBClient(r, db)
	if getErr != nil {
		return nil, nil, getErr
	}

	client, ok := services.Clients[dbClient.Name]
	if !ok {
		return nil, nil, fmt.Errorf("%w %q", ErrServiceNotFound, dbClient.Name)
	}

	return dbClient, client, nil
}

func makeClientService(db *database.DB, dbClient *model.Client) (services.Client, error) {
	module := protocols.Get(dbClient.Protocol)
	if module == nil {
		return nil, errModuleNotFound
	}

	service := module.NewClient(db, dbClient)

	return service, nil
}

//nolint:dupl //duplicate is for servers, best keep separate
func startClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		client, ok := services.Clients[dbClient.Name]
		if !ok {
			var err error
			if client, err = makeClientService(db, dbClient); handleError(w, logger, err) {
				return
			}

			services.Clients[dbClient.Name] = client
		}

		if code, _ := client.State(); code == utils.StateRunning {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Cannot start client %q, it is already running.", dbClient.Name)

			return
		}

		if err := client.Start(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

//nolint:dupl //duplicate is for servers, best keep separate
func stopClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, client, getErr := getClientService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if code, _ := client.State(); code == utils.StateOffline || code == utils.StateError {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Cannot stop client %q, it isn't running.", dbClient.Name)

			return
		}

		if err := haltService(r, client); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func restartClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, client, getErr := getClientService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := haltService(r, client); handleError(w, logger, err) {
			return
		}

		if err := client.Start(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
