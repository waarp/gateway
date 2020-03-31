package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func getLocAcc(r *http.Request, db *database.DB) (*model.LocalAgent, *model.LocalAccount, error) {
	parent, err := getLocAg(r, db)
	if err != nil {
		return nil, nil, err
	}

	login, ok := mux.Vars(r)["local_account"]
	if !ok {
		return parent, nil, &notFound{}
	}

	result := &model.LocalAccount{}
	result.LocalAgentID = parent.ID
	result.Login = login

	if err := get(db, result); err != nil {
		return parent, nil, err
	}
	return parent, result, nil
}

func getLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			_, result, err := getLocAcc(r, db)
			if err != nil {
				return err
			}

			return writeJSON(w, FromLocalAccount(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listLocalAccounts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "login ASC",
		"login+":  "login ASC",
		"login-":  "login DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			parent, err := getLocAg(r, db)
			if err != nil {
				return err
			}
			filters.Conditions = builder.Eq{"local_agent_id": parent.ID}

			var results []model.LocalAccount
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutAccount{"localAccounts": FromLocalAccounts(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			parent, err := getLocAg(r, db)
			if err != nil {
				return err
			}

			jsonAccount := &InAccount{}
			if err := readJSON(r, jsonAccount); err != nil {
				return err
			}

			account := jsonAccount.ToLocal(parent)
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

//nolint:dupl
func updateLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			agent, check, err := getLocAcc(r, db)
			if err != nil {
				return err
			}

			account := &InAccount{}
			if err := readJSON(r, account); err != nil {
				return err
			}

			if err := db.Update(account.ToLocal(agent), check.ID, false); err != nil {
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

func deleteLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			_, acc, err := getLocAcc(r, db)
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
