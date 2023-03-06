package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func ClientDBToREST(client *model.Client) *api.OutClient {
	return &api.OutClient{
		Name:         client.Name,
		Protocol:     client.Protocol,
		LocalAddress: client.LocalAddress,
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

func ClientRESTToDB(client *api.InClient) *model.Client {
	return &model.Client{
		Name:         client.Name,
		Protocol:     client.Protocol,
		LocalAddress: str(client.LocalAddress),
		ProtoConfig:  client.ProtoConfig,
	}
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
			return nil, notFound("client %q not found", clientName)
		}

		return nil, err
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

		dbClient := ClientRESTToDB(&restClient)
		if err := db.Insert(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, restClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, err := getDBClient(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(client).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func updateClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDBClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restClient := api.InClient{
			Name:         oldDBClient.Name,
			Protocol:     oldDBClient.Protocol,
			LocalAddress: &oldDBClient.LocalAddress,
			ProtoConfig:  oldDBClient.ProtoConfig,
		}
		if err := readJSON(r, &restClient); handleError(w, logger, err) {
			return
		}

		dbClient := ClientRESTToDB(&restClient)
		dbClient.ID = oldDBClient.ID

		if err := db.Update(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, restClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for a completely different type
func replaceClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDBClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restClient := api.InClient{}
		if err := readJSON(r, &restClient); handleError(w, logger, err) {
			return
		}

		dbClient := ClientRESTToDB(&restClient)
		dbClient.ID = oldDBClient.ID

		if err := db.Update(dbClient).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, restClient.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getClientService(r *http.Request, db *database.DB) (*model.Client, pipeline.Client, error) {
	dbClient, getErr := getDBClient(r, db)
	if getErr != nil {
		return nil, nil, getErr
	}

	client, ok := pipeline.Clients[dbClient.Name]
	if !ok {
		return nil, nil, errServiceNotFound
	}

	return dbClient, client, nil
}

func startClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, getErr := getDBClient(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		client, ok := pipeline.Clients[dbClient.Name]
		if !ok {
			constr, ok := constructors.ClientConstructors[dbClient.Protocol]
			if !ok {
				handleError(w, logger, errConstructorNotFound)

				return
			}

			var cliErr error
			if client, cliErr = constr(dbClient); cliErr != nil {
				handleError(w, logger, cliErr)

				return
			}

			pipeline.Clients[dbClient.Name] = client
		}

		if code, _ := client.State().Get(); code == state.Running {
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

func stopClient(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbClient, client, err := getClientService(r, db)
		if handleError(w, logger, err) {
			return
		}

		if code, _ := client.State().Get(); code == state.Offline || code == state.Error {
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
