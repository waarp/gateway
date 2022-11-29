package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl // duplicated code is about a different type
func getDBRemoteAccount(r *http.Request, db *database.DB) (*model.RemoteAgent,
	*model.RemoteAccount, error,
) {
	parent, err := retrievePartner(r, db)
	if err != nil {
		return nil, nil, err
	}

	login, ok := mux.Vars(r)["remote_account"]
	if !ok {
		return parent, nil, notFound("missing partner name")
	}

	var dbAccount model.RemoteAccount
	if err := db.Get(&dbAccount, "login=? AND remote_agent_id=?", login, parent.ID).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return parent, nil, notFound("no account '%s' found for partner %s",
				login, parent.Name)
		}

		return parent, nil, err
	}

	return parent, &dbAccount, nil
}

//nolint:dupl // duplicated code is about a different type
func listRemoteAccounts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"login", true},
		"login+":  order{"login", true},
		"login-":  order{"login", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		parent, getErr := retrievePartner(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var dbAccounts model.RemoteAccounts

		query, parseErr := parseSelectQuery(r, db, validSorting, &dbAccounts)
		if handleError(w, logger, parseErr) {
			return
		}

		query.Where("remote_agent_id=?", parent.ID)

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restAccounts, convErr := DBRemoteAccountsToREST(db, dbAccounts)
		if handleError(w, logger, convErr) {
			return
		}

		response := map[string][]*api.OutAccount{"remoteAccounts": restAccounts}
		handleError(w, logger, writeJSON(w, response))
	}
}

func getRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restAccount, convErr := DBRemoteAccountToREST(db, dbAccount)
		if handleError(w, logger, convErr) {
			return
		}

		handleError(w, logger, writeJSON(w, restAccount))
	}
}

func updateRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, oldAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restAccount := dbRemoteAccountToRESTInput(oldAccount)
		if err := readJSON(r, restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := restRemoteAccountToDB(restAccount, parent)
		dbAccount.ID = oldAccount.ID

		if err := db.Update(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, oldAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := restRemoteAccountToDB(&restAccount, parent)
		dbAccount.ID = oldAccount.ID

		if err := db.Update(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func addRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, getErr := retrievePartner(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := restRemoteAccountToDB(&restAccount, parent)
		if err := db.Insert(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getDBRemoteAccount(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(acc).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, authorizeRule(w, r, db, dbAccount))
	}
}

func revokeRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, revokeRule(w, r, db, dbAccount))
	}
}

func getRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbAccount))
	}
}

func addRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, createCrypto(w, r, db, dbAccount))
	}
}

func listRemAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbAccount))
	}
}

func deleteRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, deleteCrypto(w, r, db, dbAccount))
	}
}

func updateRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, updateCrypto(w, r, db, dbAccount))
	}
}

func replaceRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, replaceCrypto(w, r, db, dbAccount))
	}
}
