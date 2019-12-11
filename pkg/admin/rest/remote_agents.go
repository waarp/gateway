package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func createRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonAgent := &InAgent{}
			if err := readJSON(r, jsonAgent); err != nil {
				return err
			}

			agent := jsonAgent.toRemote()
			if err := db.Create(agent); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, agent.ID))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listRemoteAgents(logger *log.Logger, db *database.Db) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"proto+":  "protocol ASC",
		"proto-":  "protocol DESC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}

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

			resp := map[string][]OutAgent{"remoteAgents": fromRemoteAgents(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "remote_agent")
			if err != nil {
				return err
			}
			result := &model.RemoteAgent{ID: id}

			if err := get(db, result); err != nil {
				return err
			}

			return writeJSON(w, fromRemoteAgent(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "remote_agent")
			if err != nil {
				return &notFound{}
			}

			ag := &model.RemoteAgent{ID: id}
			if err := get(db, ag); err != nil {
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
func updateRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "remote_agent")
			if err != nil {
				return &notFound{}
			}

			if err := exist(db, &model.RemoteAgent{ID: id}); err != nil {
				return err
			}

			agent := &InAgent{}
			if err := readJSON(r, agent); err != nil {
				return err
			}

			if err := db.Update(agent.toRemote(), id, false); err != nil {
				return err
			}

			w.Header().Set("Location", location(r))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
