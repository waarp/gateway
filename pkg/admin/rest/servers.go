package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
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
			return nil, notFoundf("server %q not found", serverName)
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

		dbServer, convErr := restServerToDB(&restServer, logger)
		if handleError(w, logger, convErr) {
			return
		}

		service, srvErr := protocols.MakeServer(db, dbServer)
		if handleError(w, logger, srvErr) {
			return
		}

		if err := db.Insert(dbServer).Run(); handleError(w, logger, err) {
			return
		}

		services.Servers.Add(dbServer, service)

		w.Header().Set("Location", location(r.URL, dbServer.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func doUpdateServer(logger *log.Logger, db *database.DB, w http.ResponseWriter,
	r *http.Request, mkJSONServer func(*model.LocalAgent) *api.InServer,
) {
	oldServer, getErr := getDBServer(r, db)
	if handleError(w, logger, getErr) {
		return
	}

	if err := services.Servers.Stop(r.Context(), oldServer, false); handleError(w, logger, err) {
		return
	}

	restServer := mkJSONServer(oldServer)
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

	newService, srvErr := protocols.MakeServer(db, dbServer)
	if handleError(w, logger, srvErr) {
		return
	}

	if err := services.Servers.Restart(r.Context(), dbServer, newService); handleError(w, logger, err) {
		return
	}

	w.Header().Set("Location", locationUpdate(r.URL, dbServer.Name))
	w.WriteHeader(http.StatusCreated)
}

func updateServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doUpdateServer(logger, db, w, r, func(dbServer *model.LocalAgent) *api.InServer {
			return &api.InServer{
				Name:          asNullable(dbServer.Name),
				Protocol:      asNullable(dbServer.Protocol),
				Address:       asNullable(dbServer.Address.String()),
				RootDir:       asNullable(dbServer.RootDir),
				ReceiveDir:    asNullable(dbServer.ReceiveDir),
				SendDir:       asNullable(dbServer.SendDir),
				TmpReceiveDir: asNullable(dbServer.TmpReceiveDir),
				ProtoConfig:   api.UpdateObject[any](dbServer.ProtoConfig),
			}
		})
	}
}

func replaceServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doUpdateServer(logger, db, w, r, func(*model.LocalAgent) *api.InServer {
			return &api.InServer{}
		})
	}
}

//nolint:dupl //duplicate is for clients, best keep separate
func deleteServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := services.Servers.Stop(r.Context(), dbServer, true); handleError(w, logger, err) {
			return
		}

		if err := db.Delete(dbServer).Run(); handleError(w, logger, err) {
			return
		}

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

		handleError(w, logger, addCredential(w, r, db, dbServer))
	}
}

func getServerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, err := getDBServer(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, getCredential(w, r, db, dbServer))
	}
}

func removeServerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, err := getDBServer(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, removeCredential(w, r, db, dbServer))
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

//nolint:dupl //duplicate is for clients, best keep separate
func stopServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := services.Servers.Stop(r.Context(), dbServer, false); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

//nolint:dupl //duplicate is for clients, best keep separate
func startServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := services.Servers.Start(dbServer); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func restartServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		newService, srvErr := protocols.MakeServer(db, dbServer)
		if handleError(w, logger, srvErr) {
			return
		}

		if err := services.Servers.Restart(r.Context(), dbServer,
			newService); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
