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

func getServ(r *http.Request, db *database.DB) (*model.LocalAgent, error) {
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

func getServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, FromLocalAgent(result, rules))
		handleError(w, logger, err)
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
	typ := (&model.LocalAgent{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		var servers model.LocalAgents

		query, err := parseSelectQuery(r, db, validSorting, &servers)
		if handleError(w, logger, err) {
			return
		}

		query.Where("owner=?", conf.GlobalConfig.GatewayName)

		if err2 := parseProtoParam(r, query); handleError(w, logger, err2) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		ids := make([]uint64, len(servers))
		for i := range servers {
			ids[i] = servers[i].ID
		}

		rules, err := getAuthorizedRuleList(db, typ, ids)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutServer{"servers": FromLocalAgents(servers, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

//nolint:dupl // duplicated code is about a different type
func addServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var serv api.InServer
		if err := readJSON(r, &serv); handleError(w, logger, err) {
			return
		}

		var agent model.LocalAgent

		servToDB(logger, &serv, &agent)

		if err := db.Insert(&agent).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, agent.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl // duplicated code is about a different type
func updateServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jServ api.InServer
		if err := readJSON(r, &jServ); handleError(w, logger, err) {
			return
		}

		servToDB(logger, &jServ, agent)

		if err := db.Update(agent).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, agent.Name))
		w.WriteHeader(http.StatusCreated)

		if jServ.Name != nil || jServ.Protocol != nil || jServ.Address != nil {
			fmt.Fprint(w, ServerRestartRequiredMsg)
		}
	}
}

func replaceServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jServ api.InServer
		if err := readJSON(r, &jServ); handleError(w, logger, err) {
			return
		}

		agent := &model.LocalAgent{ID: old.ID}
		servToDB(logger, &jServ, agent)

		if err := db.Update(agent).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, agent.Name))
		w.WriteHeader(http.StatusCreated)

		if jServ.Name != nil || jServ.Protocol != nil || jServ.Address != nil {
			fmt.Fprint(w, ServerRestartRequiredMsg)
		}
	}
}

func deleteServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(ag).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = authorizeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func revokeServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = revokeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func getServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = getCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func addServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCrypto(w, r, db, ag.TableName(), ag.ID)
		if handleError(w, logger, err) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func listServerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCryptos(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func deleteServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCrypto(w, r, db, ag.TableName(), ag.ID)
		if handleError(w, logger, err) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func updateServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCrypto(w, r, db, ag.TableName(), ag.ID)
		if handleError(w, logger, err) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func replaceServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCrypto(w, r, db, ag.TableName(), ag.ID)
		if handleError(w, logger, err) {
			return
		}

		fmt.Fprint(w, ServerCertRestartRequiredMsg)
	}
}

func enableServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return enableDisableServer(logger, db, true)
}

func disableServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return enableDisableServer(logger, db, false)
}

func enableDisableServer(logger *log.Logger, db *database.DB, enable bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		if ag.Enabled == enable {
			w.WriteHeader(http.StatusAccepted)

			return // nothing to do
		}

		ag.Enabled = enable

		if handleError(w, logger, db.Update(ag).Cols("enabled").Run()) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func getProtoService(r *http.Request, protoServices map[uint64]proto.Service,
	db *database.DB,
) (*model.LocalAgent, proto.Service, error) {
	dbServer, err := getServ(r, db)
	if err != nil {
		return nil, nil, err
	}

	service, ok := protoServices[dbServer.ID]
	if !ok {
		return nil, nil, errServiceNotFound
	}

	return dbServer, service, nil
}

func haltServer(r *http.Request, serv proto.Service) error {
	const haltTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(r.Context(), haltTimeout)
	defer cancel()

	if err := serv.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

func stopServer(protoServices map[uint64]proto.Service) handler {
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

			if err := haltServer(r, service); handleError(w, logger, err) {
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	}
}

var errConstructorNotFound = errors.New("could not instantiate the service: protocol not found")

func startServer(protoServices map[uint64]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dbServer, err := getServ(r, db)
			if handleError(w, logger, err) {
				return
			}

			service, ok := protoServices[dbServer.ID]
			if !ok {
				constr, ok := constructors.ServiceConstructors[dbServer.Protocol]
				if !ok {
					handleError(w, logger, errConstructorNotFound)

					return
				}

				servLogger := conf.GetLogger(dbServer.Name)
				service = constr(db, servLogger)
				protoServices[dbServer.ID] = service
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

func restartServer(protoServices map[uint64]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			dbServer, service, err := getProtoService(r, protoServices, db)
			if handleError(w, logger, err) {
				return
			}

			if err := haltServer(r, service); handleError(w, logger, err) {
				return
			}

			if err := service.Start(dbServer); handleError(w, logger, err) {
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	}
}
