package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func getLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "local_account")
			if err != nil {
				return err
			}
			result := &model.LocalAccount{ID: id}

			if err := get(db, result); err != nil {
				return err
			}

			return writeJSON(w, fromLocal(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listLocalAccounts(logger *log.Logger, db *database.Db) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "login ASC",
		"agent+":  "local_agent_id ASC",
		"agent-":  "local_agent_id DESC",
		"login+":  "login ASC",
		"login-":  "login DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			if err := parseAgentParam(r, filters, "local_agent_id"); err != nil {
				return err
			}

			var results []model.LocalAccount
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutAccount{"localAccounts": fromLocalArray(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonAccount := &InAccount{}
			if err := readJSON(r, jsonAccount); err != nil {
				return err
			}

			account := jsonAccount.toLocal()
			if err := db.Create(account); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, account.ID))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func updateLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "local_account")
			if err != nil {
				return &notFound{}
			}

			if err := exist(db, &model.LocalAccount{ID: id}); err != nil {
				return err
			}

			account := &InAccount{}
			if err := readJSON(r, account); err != nil {
				return err
			}

			if err := db.Update(account.toLocal(), id, false); err != nil {
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

func deleteLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "local_account")
			if err != nil {
				return &notFound{}
			}

			acc := &model.LocalAccount{ID: id}
			if err := get(db, acc); err != nil {
				return err
			}

			if err := db.Delete(acc); err != nil {
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
