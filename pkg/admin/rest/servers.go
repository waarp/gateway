package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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

		agent := servToDB(&serv, 0, logger)
		if err := db.Insert(agent).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, agent.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl // duplicated code is about a different type
func updateServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		jServ := newInServer(old)
		if err := readJSON(r, jServ); handleError(w, logger, err) {
			return
		}

		serv := servToDB(jServ, old.ID, logger)
		if err := db.Update(serv).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, serv.Name))
		w.WriteHeader(http.StatusCreated)
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

		serv := servToDB(&jServ, old.ID, logger)
		if err := db.Update(serv).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, serv.Name))
		w.WriteHeader(http.StatusCreated)
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
		handleError(w, logger, err)
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
		handleError(w, logger, err)
	}
}

func updateServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func replaceServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}
