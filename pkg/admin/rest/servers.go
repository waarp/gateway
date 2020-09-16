package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func getLocAg(r *http.Request, db *database.DB) (*model.LocalAgent, error) {
	agentName, ok := mux.Vars(r)["server"]
	if !ok {
		return nil, notFound("missing server name")
	}
	agent := &model.LocalAgent{Name: agentName, Owner: database.Owner}
	if err := db.Get(agent); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("server '%s' not found", agentName)
		}
		return nil, err
	}
	return agent, nil
}

func getServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getLocAg(r, db)
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
	validSorting := map[string]string{
		"default": "name ASC",
		"proto+":  "protocol ASC",
		"proto-":  "protocol DESC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}
	typ := (&model.LocalAgent{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		filters, err := parseListFilters(r, validSorting)
		if handleError(w, logger, err) {
			return
		}
		filters.Conditions = builder.Eq{"owner": database.Owner}
		if err := parseProtoParam(r, filters); handleError(w, logger, err) {
			return
		}

		var results []model.LocalAgent
		if err := db.Select(&results, filters); handleError(w, logger, err) {
			return
		}

		ids := make([]uint64, len(results))
		for i, res := range results {
			ids[i] = res.ID
		}
		rules, err := getAuthorizedRuleList(db, typ, ids)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutServer{"servers": FromLocalAgents(results, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func addServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var serv api.InServer
		if err := readJSON(r, &serv); handleError(w, logger, err) {
			return
		}

		agent := servToDB(&serv, 0)
		if err := db.Create(agent); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, agent.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		serv := newInServer(old)
		if err := readJSON(r, serv); handleError(w, logger, err) {
			return
		}

		if err := db.Update(servToDB(serv, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(serv.Name)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		var serv api.InServer
		if err := readJSON(r, &serv); handleError(w, logger, err) {
			return
		}

		if err := db.Update(servToDB(&serv, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(serv.Name)))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(ag); handleError(w, logger, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = authorizeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func revokeServer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = revokeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func getServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = getCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func addServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func listServerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCertificates(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func deleteServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func updateServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func replaceServerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}
