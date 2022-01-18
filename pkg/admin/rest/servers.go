package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const serviceShutdownTimeout = 5 * time.Second

//nolint:dupl //duplicate is for a completely different type (users), keep separate
func getDBServer(r *http.Request, db *database.DB) (*model.LocalAgent, error) {
	serverName, ok := mux.Vars(r)["server"]
	if !ok {
		return nil, notFound("missing server name")
	}

	var serv model.LocalAgent
	if err := db.Get(&serv, "name=? AND owner=?", serverName, conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("server '%s' not found", serverName)
		}

		return nil, fmt.Errorf("failed to retrieve server %q: %w", serverName, err)
	}

	return &serv, nil
}

func getServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restServer, convErr := DBServerToREST(db, dbServer)
		if handleError(w, logger, convErr) {
			return
		}

		handleError(w, logger, writeJSON(w, restServer))
	}
}

func listServers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"proto+":  order{"protocol", true},
		"proto-":  order{"protocol", false},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbServers model.LocalAgents

		query, parseErr := parseSelectQuery(r, db, validSorting, &dbServers)
		if handleError(w, logger, parseErr) {
			return
		}

		query.Where("owner=?", conf.GlobalConfig.GatewayName)

		if err := parseProtoParam(r, query); handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restServers, convErr := DBServersToREST(db, dbServers)
		if handleError(w, logger, convErr) {
			return
		}

		response := map[string][]*api.OutServer{"servers": restServers}
		handleError(w, logger, writeJSON(w, response))
	}
}

func addServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restServer api.InServer
		if err := readJSON(r, &restServer); handleError(w, logger, err) {
			return
		}

		dbServer, err := restServerToDB(&restServer, logger)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(dbServer).Run(); handleError(w, logger, err) {
			return
		}

		module := protocols.Get(dbServer.Protocol)

		server := module.NewServer(db, dbServer)
		if err := server.Start(); handleError(w, logger, err) {
			return
		}

		services.Servers[dbServer.Name] = server

		w.Header().Set("Location", location(r.URL, dbServer.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldServer, oldService, getErr := getServerService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		oldName := oldServer.Name
		restServer := &api.InServer{
			Name:          api.AsNullable(oldServer.Name),
			Protocol:      api.AsNullable(oldServer.Protocol),
			Address:       api.AsNullable(oldServer.Address.String()),
			RootDir:       api.AsNullable(oldServer.RootDir),
			ReceiveDir:    api.AsNullable(oldServer.ReceiveDir),
			SendDir:       api.AsNullable(oldServer.SendDir),
			TmpReceiveDir: api.AsNullable(oldServer.TmpReceiveDir),
			ProtoConfig:   oldServer.ProtoConfig,
		}

		if err := readJSON(r, restServer); handleError(w, logger, err) {
			return
		}

		dbServer, convErr := restServerToDB(restServer, logger)
		if handleError(w, logger, convErr) {
			return
		}

		dbServer.ID = oldServer.ID

		if err := db.Update(dbServer).Run(); handleError(w, logger, err) {
			return
		}

		newService, servErr := makeServerService(db, dbServer)
		if handleError(w, logger, servErr) {
			return
		}

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

		delete(services.Servers, oldName)
		services.Servers[dbServer.Name] = newService

		w.Header().Set("Location", locationUpdate(r.URL, dbServer.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldServer, oldService, getErr := getServerService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restServer api.InServer
		if err := readJSON(r, &restServer); handleError(w, logger, err) {
			return
		}

		dbServer, convErr := restServerToDB(&restServer, logger)
		if handleError(w, logger, convErr) {
			return
		}

		dbServer.ID = oldServer.ID

		if err := db.Update(dbServer).Run(); handleError(w, logger, err) {
			return
		}

		newService, servErr := makeServerService(db, dbServer)
		if handleError(w, logger, servErr) {
			return
		}

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

		delete(services.Servers, oldServer.Name)
		services.Servers[dbServer.Name] = newService

		w.Header().Set("Location", locationUpdate(r.URL, dbServer.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for clients, best keep separate
func deleteServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, service, err := getServerService(r, db)
		if handleError(w, logger, err) {
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

		if err := db.Delete(dbServer).Run(); handleError(w, logger, err) {
			return
		}

		delete(services.Servers, dbServer.Name)

		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, authorizeRule(w, r, db, dbServer))
	}
}

func revokeServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, revokeRule(w, r, db, dbServer))
	}
}

func addServerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, err := getDBServer(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, addCredential(w, r, db, dbServer, dbServer.Protocol))
	}
}

func removeServerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getDBServer(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, removeCredential(w, r, db, ag))
	}
}

// Deprecated: replaced by Credentials.
func getServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbServer))
	}
}

// Deprecated: replaced by Credentials.
func addServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, createCrypto(w, r, db, dbServer)) {
			return
		}
	}
}

// Deprecated: replaced by Credentials.
func listServerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbServer))
	}
}

// Deprecated: replaced by Credentials.
func deleteServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, deleteCrypto(w, r, db, dbServer)) {
			return
		}
	}
}

// Deprecated: replaced by Credentials.
func updateServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, updateCrypto(w, r, db, dbServer)) {
			return
		}
	}
}

// Deprecated: replaced by Credentials.
func replaceServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, replaceCrypto(w, r, db, dbServer)) {
			return
		}
	}
}

func enableServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return enableDisableServer(logger, db, false)
}

func disableServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return enableDisableServer(logger, db, true)
}

func enableDisableServer(logger *log.Logger, db *database.DB, disable bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if dbServer.Disabled == disable {
			w.WriteHeader(http.StatusAccepted)

			return // nothing to do
		}

		dbServer.Disabled = disable

		if handleError(w, logger, db.Update(dbServer).Cols("disabled").Run()) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

var ErrServiceNotFound = errors.New("service not found")

func getServerService(r *http.Request, db *database.DB,
) (*model.LocalAgent, services.Server, error) {
	dbServer, getErr := getDBServer(r, db)
	if getErr != nil {
		return nil, nil, getErr
	}

	service, ok := services.Servers[dbServer.Name]
	if !ok {
		return nil, nil, fmt.Errorf("%w %q", ErrServiceNotFound, dbServer.Name)
	}

	return dbServer, service, nil
}

func haltService(r *http.Request, serv services.Service) error {
	const haltTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(r.Context(), haltTimeout)
	defer cancel()

	if err := serv.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

//nolint:dupl //duplicate is for clients, best keep separate
func stopServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, service, err := getServerService(r, db)
		if handleError(w, logger, err) {
			return
		}

		if code, _ := service.State(); code == utils.StateOffline || code == utils.StateError {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Cannot stop server %q, it isn't running.", dbServer.Name)

			return
		}

		if err := haltService(r, service); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

var errModuleNotFound = errors.New("could not instantiate the service: protocol not found")

func makeServerService(db *database.DB, dbServer *model.LocalAgent) (services.Server, error) {
	module := protocols.Get(dbServer.Protocol)
	if module == nil {
		return nil, errModuleNotFound
	}

	service := module.NewServer(db, dbServer)

	return service, nil
}

//nolint:dupl //duplicate is for clients, best keep separate
func startServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		service, ok := services.Servers[dbServer.Name]
		if !ok {
			var err error

			service, err = makeServerService(db, dbServer)
			if handleError(w, logger, err) {
				return
			}

			services.Servers[dbServer.Name] = service
		}

		if code, _ := service.State(); code == utils.StateRunning {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Cannot start server %q, it is already running.", dbServer.Name)

			return
		}

		if err := service.Start(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func restartServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, service, getErr := getServerService(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := haltService(r, service); handleError(w, logger, err) {
			return
		}

		if err := service.Start(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
