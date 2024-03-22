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
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	ServerRestartRequiredMsg = "A restart is required when changing a server's " +
		"name, protocol or address for the changes to be effective."
	ServerCertRestartRequiredMsg = "A restart is required when changing a server's " +
		"certificates for the changes to be effective."
)

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

		return nil, err
	}

	return &serv, nil
}

func updateServiceName(oldName, newName string, m map[string]proto.Service) error {
	service, ok := m[oldName]
	if !ok {
		return internal("service %q not found", oldName)
	}

	delete(m, oldName)
	m[newName] = service

	return nil
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

func addServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var restServer api.InServer
			if err := readJSON(r, &restServer); handleError(w, logger, err) {
				return
			}

			dbServer := restServerToDB(&restServer, logger)
			if err := db.Insert(dbServer).Run(); handleError(w, logger, err) {
				return
			}

			constr := constructors.ServiceConstructors[dbServer.Protocol]
			serverLog := conf.GetLogger(dbServer.Name)

			protoServices[dbServer.Name] = constr(db, serverLog)

			w.Header().Set("Location", location(r.URL, dbServer.Name))
			w.WriteHeader(http.StatusCreated)
		}
	}
}

func updateServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			oldServer, getErr := getDBServer(r, db)
			if handleError(w, logger, getErr) {
				return
			}

			oldName := oldServer.Name
			restServer := dbServerToRESTInput(oldServer)

			if err := readJSON(r, restServer); handleError(w, logger, err) {
				return
			}

			dbServer := restServerToDB(restServer, logger)
			dbServer.ID = oldServer.ID

			if err := db.Update(dbServer).Run(); handleError(w, logger, err) {
				return
			}

			if err := updateServiceName(oldName, dbServer.Name,
				protoServices); handleError(w, logger, err) {
				return
			}

			w.Header().Set("Location", locationUpdate(r.URL, dbServer.Name))
			w.WriteHeader(http.StatusCreated)

			if restServer.Name != nil || restServer.Protocol != nil || restServer.Address != nil {
				fmt.Fprint(w, ServerRestartRequiredMsg)
			}
		}
	}
}

func replaceServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			oldServer, getErr := getDBServer(r, db)
			if handleError(w, logger, getErr) {
				return
			}

			var restServer api.InServer
			if err := readJSON(r, &restServer); handleError(w, logger, err) {
				return
			}

			dbServer := restServerToDB(&restServer, logger)
			dbServer.ID = oldServer.ID

			if err := db.Update(dbServer).Run(); handleError(w, logger, err) {
				return
			}

			if err := updateServiceName(oldServer.Name, dbServer.Name,
				protoServices); handleError(w, logger, err) {
				return
			}

			w.Header().Set("Location", locationUpdate(r.URL, dbServer.Name))
			w.WriteHeader(http.StatusCreated)

			if restServer.Name != nil || restServer.Protocol != nil || restServer.Address != nil {
				fmt.Fprint(w, ServerRestartRequiredMsg)
			}
		}
	}
}

func deleteServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dbServer, service, err := getProtoService(r, protoServices, db)
			if handleError(w, logger, err) {
				return
			}

			switch code, _ := service.State().Get(); code {
			case state.Error, state.Offline:
			default:
				handleError(w, logger, badRequest("cannot delete an active server, "+
					"it must be shut down first"))

				return
			}

			delete(protoServices, dbServer.Name)

			if err := db.Delete(dbServer).Run(); handleError(w, logger, err) {
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
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

func getServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbServer))
	}
}

func addServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, createCrypto(w, r, db, dbServer)) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func listServerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbServer))
	}
}

func deleteServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, deleteCrypto(w, r, db, dbServer)) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func updateServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, updateCrypto(w, r, db, dbServer)) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func replaceServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbServer, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if handleError(w, logger, replaceCrypto(w, r, db, dbServer)) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
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

func getProtoService(r *http.Request, protoServices map[string]proto.Service,
	db *database.DB,
) (*model.LocalAgent, proto.Service, error) {
	dbServer, getErr := getDBServer(r, db)
	if getErr != nil {
		return nil, nil, getErr
	}

	service, ok := protoServices[dbServer.Name]
	if !ok {
		return nil, nil, errServiceNotFound
	}

	return dbServer, service, nil
}

func haltService(r *http.Request, serv proto.Stopper) error {
	const haltTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(r.Context(), haltTimeout)
	defer cancel()

	if err := serv.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

func stopServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dbServer, service, err := getProtoService(r, protoServices, db)
			if handleError(w, logger, err) {
				return
			}

			if code, _ := service.State().Get(); code == state.Offline || code == state.Error {
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
}

var errConstructorNotFound = errors.New("could not instantiate the service: protocol not found")

func startServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dbServer, getErr := getDBServer(r, db)
			if handleError(w, logger, getErr) {
				return
			}

			service, ok := protoServices[dbServer.Name]
			if !ok {
				constr, ok := constructors.ServiceConstructors[dbServer.Protocol]
				if !ok {
					handleError(w, logger, errConstructorNotFound)

					return
				}

				servLogger := conf.GetLogger(dbServer.Name)
				service = constr(db, servLogger)
				protoServices[dbServer.Name] = service
			}

			if code, _ := service.State().Get(); code == state.Running {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Cannot start server %q, it is already running.", dbServer.Name)

				return
			}

			if err := service.Start(dbServer); handleError(w, logger, err) {
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func restartServer(protoServices map[string]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dbServer, service, getErr := getProtoService(r, protoServices, db)
			if handleError(w, logger, getErr) {
				return
			}

			if err := haltService(r, service); handleError(w, logger, err) {
				return
			}

			if err := service.Start(dbServer); handleError(w, logger, err) {
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	}
}
