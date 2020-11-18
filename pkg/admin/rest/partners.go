package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getRemAg(r *http.Request, db *database.DB) (*model.RemoteAgent, error) {
	agentName, ok := mux.Vars(r)["remote_agent"]
	if !ok {
		return nil, notFound("missing partner name")
	}
	agent := &model.RemoteAgent{Name: agentName}
	if err := db.Get(agent); err != nil {
		if err == database.ErrNotFound {
			return nil, notFound("partner '%s' not found", agentName)
		}
		return nil, err
	}
	return agent, nil
}

func createRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			part := &api.InPartner{}
			if err := readJSON(r, part); err != nil {
				return err
			}

			agent := partToDB(part, 0)
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

func listRemoteAgents(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"proto+":  "protocol ASC",
		"proto-":  "protocol DESC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}
	typ := (&model.RemoteAgent{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			if err := parseProtoParam(r, filters); err != nil {
				return err
			}

			var results []model.RemoteAgent
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

			resp := map[string][]api.OutPartner{"partners": FromRemoteAgents(results, rules)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getRemAg(r, db)
			if err != nil {
				return err
			}

			rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
			if err != nil {
				return err
			}

			return writeJSON(w, FromRemoteAgent(result, rules))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func updateRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getRemAg(r, db)
			if err != nil {
				return err
			}

			part := newInPartner(old)
			if err := readJSON(r, part); err != nil {
				return err
			}

			if err := db.Update(partToDB(part, old.ID)); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, str(part.Name)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func replaceRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getRemAg(r, db)
			if err != nil {
				return err
			}

			part := &api.InPartner{}
			if err := readJSON(r, part); err != nil {
				return err
			}

			if err := db.Update(partToDB(part, old.ID)); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, str(part.Name)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func authorizeRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func revokeRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func getRemAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func createRemAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func listRemAgentCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func deleteRemAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func updateRemAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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

func replaceRemAgentCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ag, err := getRemAg(r, db)
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
