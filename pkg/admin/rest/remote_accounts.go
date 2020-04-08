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
	parent, err := getRemAg(r, db)
	if err != nil {
		return nil, nil, err
	}

	login, ok := mux.Vars(r)["remote_account"]
	if !ok {
		return parent, nil, notFound("missing partner name")
	}

	result := &model.RemoteAccount{}
	result.RemoteAgentID = parent.ID
	result.Login = login

	if err := db.Get(result); err != nil {
		if err == database.ErrNotFound {
			return parent, nil, notFound("no account '%s' found for partner %s",
				login, parent.Name)
		}
		return parent, nil, err
	}
	return parent, result, nil
}

//nolint:dupl
func listRemoteAccounts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "login ASC",
		"login+":  "login ASC",
		"login-":  "login DESC",
	}
	typ := (&model.RemoteAccount{}).TableName()

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

			ids := make([]uint64, len(results))
			for i, res := range results {
				ids[i] = res.ID
			}
			rules, err := getAuthorizedRuleList(db, typ, ids)
			if err != nil {
				return err
			}

			resp := map[string][]OutAccount{"remoteAccounts": FromRemoteAccounts(results, rules)}
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

			rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
			if err != nil {
				return err
			}

			return writeJSON(w, FromRemoteAccount(result, rules))
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

			w.Header().Set("Location", location(r, account.Login))
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
