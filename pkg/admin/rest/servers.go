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
	agentName, ok := mux.Vars(r)["local_agent"]
	if !ok {
		return nil, notFound("missing server name")
	}
	agent := &model.LocalAgent{Name: agentName, Owner: database.Owner}
	if err := db.Get(agent); err != nil {
		if err == database.ErrNotFound {
			return nil, notFound("server '%s' not found", agentName)
		}
		return nil, err
	}
	return agent, nil
}

func getLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
			if err != nil {
				return err
			}

			return writeJSON(w, FromLocalAgent(result, rules))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listLocalAgents(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"proto+":  "protocol ASC",
		"proto-":  "protocol DESC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}
	typ := (&model.LocalAgent{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			filters.Conditions = builder.Eq{"owner": database.Owner}
			if err := parseProtoParam(r, filters); err != nil {
				return err
			}

			var results []model.LocalAgent
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			ids := make([]uint64, len(results))
			for i, res := range results {
				ids[i] = res.ID
			}
			rules, err := getAuthorizedRuleList(db, typ, ids)
			if err != nil {
				return err
			}

			resp := map[string][]api.OutServer{"servers": FromLocalAgents(results, rules)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			serv := &api.InServer{}
			if err := readJSON(r, serv); err != nil {
				return err
			}

			agent := servToDB(serv, 0)
			if err := db.Create(agent); err != nil {
				return err
			}

			w.Header().Set("Location", location(r.URL, agent.Name))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func updateLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			serv := newInServer(old)
			if err := readJSON(r, serv); err != nil {
				return err
			}

			if err := db.Update(servToDB(serv, old.ID)); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, str(serv.Name)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func replaceLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			serv := &api.InServer{}
			if err := readJSON(r, serv); err != nil {
				return err
			}

			if err := db.Update(servToDB(serv, old.ID)); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, str(serv.Name)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			if err := db.Delete(ag); err != nil {
				return err
			}
			w.WriteHeader(http.StatusNoContent)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func authorizeLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return authorizeRule(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func revokeLocalAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return revokeRule(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getLocAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return getCertificate(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createLocAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return createCertificate(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listLocAgentCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return listCertificates(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteLocAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return deleteCertificate(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func updateLocAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return updateCertificate(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func replaceLocAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			return replaceCertificate(w, r, db, ag.TableName(), ag.ID)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
