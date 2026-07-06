package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
)

func ClientDBToREST(client *model.Client) *api.OutClient {
	return &api.OutClient{
		Name:                 client.Name,
		Enabled:              !client.Disabled,
		Protocol:             client.Protocol,
		LocalAddress:         client.LocalAddress.String(),
		NbOfAttempts:         client.NbOfAttempts,
		FirstRetryDelay:      client.FirstRetryDelay,
		RetryIncrementFactor: client.RetryIncrementFactor,
		ProtoConfig:          client.ProtoConfig,
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
	cli := &model.Client{}

	setIfValid(&cli.Name, client.Name)
	setIfValid(&cli.Protocol, client.Protocol)
	setIfValid(&cli.Disabled, client.Disabled)
	setIfValid(&cli.NbOfAttempts, client.NbOfAttempts)
	setIfValid(&cli.FirstRetryDelay, client.FirstRetryDelay)
	setIfValid(&cli.RetryIncrementFactor, client.RetryIncrementFactor)

	if client.ProtoConfig != nil {
		cli.ProtoConfig = model.Map[any](client.ProtoConfig)
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
	if err := db.Get(&client, "name=?", clientName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("client %q not found", clientName)
		}

		return nil, fmt.Errorf("failed to retrieve client %q: %w", clientName, err)
	}

	if !services.Clients.Exists(client) {
		service, err := protocols.MakeClient(db, &client)
		if err != nil {
			return nil, fmt.Errorf("failed to make client service %q: %w", clientName, err)
		}

		services.Clients.Add(&client, service)
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
	//nolint:goconst //too specific, keep separate
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

		service, mkErr := protocols.MakeClient(db, dbClient)
		if handleError(w, logger, mkErr) {
			return
		}

		if err := db.Insert(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		services.Clients.Add(dbClient, service)

		w.Header().Set("Location", location(r.URL, dbClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for servers, best keep separate
func deleteClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		if err := services.Clients.Remove(r.Context(), dbClient); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func doUpdateClient(logger *log.Logger, db *database.DB, w http.ResponseWriter,
	r *http.Request, mkJSONClient func(*model.Client) *api.InClient,
) {
	oldClient, getErr := getDBClient(r, db)
	if handleError(w, logger, getErr) {
		return
	}

	stopped, stopErr := services.Clients.Stop(r.Context(), oldClient)
	if handleError(w, logger, stopErr) {
		return
	}

	restClient := mkJSONClient(oldClient)
	if err := readJSON(r, restClient); handleError(w, logger, err) {
		return
	}

	dbClient, convErr := ClientRESTToDB(restClient)
	if handleError(w, logger, convErr) {
		return
	}

	dbClient.ID = oldClient.ID

	if err := db.Update(dbClient).Run(); handleError(w, logger, err) {
		return
	}

	if stopped {
		if _, err := services.Clients.Start(dbClient); handleError(w, logger, err) {
			return
		}
	}

	w.Header().Set("Location", locationUpdate(r.URL, dbClient.Name))
	w.WriteHeader(http.StatusCreated)
}

func updateClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doUpdateClient(logger, db, w, r, func(dbClient *model.Client) *api.InClient {
			return &api.InClient{
				Name:                 asNullable(dbClient.Name),
				Protocol:             asNullable(dbClient.Protocol),
				Disabled:             asNullableBool(dbClient.Disabled),
				LocalAddress:         asNullable(dbClient.LocalAddress.String()),
				NbOfAttempts:         asNullable(dbClient.NbOfAttempts),
				FirstRetryDelay:      asNullable(dbClient.FirstRetryDelay),
				RetryIncrementFactor: asNullable(dbClient.RetryIncrementFactor),
				ProtoConfig:          api.UpdateObject[any](dbClient.ProtoConfig),
			}
		})
	}
}

//nolint:dupl //duplicate is for a completely different type
func replaceClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doUpdateClient(logger, db, w, r, func(*model.Client) *api.InClient {
			return &api.InClient{}
		})
	}
}

//nolint:dupl //duplicate is for servers, best keep separate
func startClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if _, err := services.Clients.Stop(r.Context(), dbClient); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func stopClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return restartClient(logger, db)
}

func restartClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := services.Clients.Restart(r.Context(), dbClient); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
