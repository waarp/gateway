package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func getRemAcc(r *http.Request, db *database.DB) (*model.RemoteAgent, *model.RemoteAccount, error) {
	result := &model.RemoteAccount{}
	parent, err := getRemAg(r, db)
	if err != nil {
		return nil, nil, err
	}

	result.RemoteAgentID = parent.ID
	result.Login = mux.Vars(r)["remote_account"]

	if err := get(db, result); err != nil {
		return nil, nil, err
	}
	return parent, result, nil
}

func listRemoteAccounts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "login ASC",
		"agent+":  "remote_agent_id ASC",
		"agent-":  "remote_agent_id DESC",
		"login+":  "login ASC",
		"login-":  "login DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			parent, err := getRemAg(r, db)
			if err != nil {
				return err
			}
			filters.Conditions = builder.Eq{"remote_agent_id": parent.ID}

			var results []model.RemoteAccount
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutAccount{"remoteAccounts": FromRemoteAccounts(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			_, result, err := getRemAcc(r, db)
			if err != nil {
				return err
			}

			return writeJSON(w, FromRemoteAccount(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func updateRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			agent, check, err := getRemAcc(r, db)
			if err != nil {
				return err
			}

			account := &InAccount{}
			if err := readJSON(r, account); err != nil {
				return err
			}

			if err := db.Update(account.ToRemote(agent), check.ID, false); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r, account.Login, check.Login))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			parent, err := getRemAg(r, db)
			if err != nil {
				return err
			}

			jsonAccount := &InAccount{}
			if err := readJSON(r, jsonAccount); err != nil {
				return err
			}

			account := jsonAccount.ToRemote(parent)
			if err := db.Create(account); err != nil {
				return err
			}

			w.Header().Set("Location", location2(r, account.Login))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			_, acc, err := getRemAcc(r, db)
			if err != nil {
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
