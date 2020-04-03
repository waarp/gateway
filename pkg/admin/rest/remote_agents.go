package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getRemAg(r *http.Request, db *database.DB) (*model.RemoteAgent, error) {
	agentName, ok := mux.Vars(r)["remote_agent"]
	if !ok {
		return nil, &notFound{}
	}
	agent := &model.RemoteAgent{Name: agentName}
	if err := get(db, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func createRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonAgent := &InRemoteAgent{}
			if err := readJSON(r, jsonAgent); err != nil {
				return err
			}

			agent := jsonAgent.ToModel()
			if err := db.Create(agent); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, agent.Name))
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

			resp := map[string][]OutRemoteAgent{"remoteAgents": FromRemoteAgents(results, rules)}
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

//nolint:dupl
func updateRemoteAgent(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getRemAg(r, db)
			if err != nil {
				return err
			}

			agent := &InRemoteAgent{}
			if err := readJSON(r, agent); err != nil {
				return err
			}

			if err := db.Update(agent.ToModel(), check.ID, false); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r, agent.Name, check.Name))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
