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
func getDBLocalAccount(r *http.Request, db *database.DB) (*model.LocalAgent,
	*model.LocalAccount, error,
) {
	parent, err := getDBServer(r, db)
	if err != nil {
		return nil, nil, err
	}

	login, ok := mux.Vars(r)["local_account"]
	if !ok {
		return parent, nil, notFound("missing account login")
	}

	var dbAccount model.LocalAccount
	if err := db.Get(&dbAccount, "login=? AND local_agent_id=?", login, parent.ID).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return parent, nil, notFound("no account '%s' found for server %s",
				login, parent.Name)
		}

		return parent, nil, err
	}

	return parent, &dbAccount, nil
}

func getLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, err := getDBLocalAccount(r, db)
		if handleError(w, logger, err) {
			return
		}

		restAccount, err := DBLocalAccountToREST(db, dbAccount)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, restAccount))
	}
}

//nolint:dupl // duplicated code is about a different type
func listLocalAccounts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"login", true},
		"login+":  order{"login", true},
		"login-":  order{"login", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		parent, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var dbAccounts model.LocalAccounts

		query, getErr := parseSelectQuery(r, db, validSorting, &dbAccounts)
		if handleError(w, logger, getErr) {
			return
		}

		query.Where("local_agent_id=?", parent.ID)

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restAccounts, getErr := DBLocalAccountsToRest(db, dbAccounts)
		if handleError(w, logger, getErr) {
			return
		}

		response := map[string][]*api.OutAccount{"localAccounts": restAccounts}
		handleError(w, logger, writeJSON(w, response))
	}
}

func addLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, getErr := getDBServer(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount, convErr := restLocalAccountToDB(&restAccount, parent)
		if handleError(w, logger, convErr) {
			return
		}

		if err := db.Insert(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, oldAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restAccount := dbLocalAccountToRESTInput(oldAccount)
		if err := readJSON(r, restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount, convErr := restLocalAccountToDB(restAccount, parent)
		if handleError(w, logger, convErr) {
			return
		}

		dbAccount.ID = oldAccount.ID
		if err := db.Update(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, oldAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount, convErr := restLocalAccountToDB(&restAccount, parent)
		if handleError(w, logger, convErr) {
			return
		}

		dbAccount.ID = oldAccount.ID
		if err := db.Update(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(dbAccount).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, authorizeRule(w, r, db, dbAccount))
	}
}

func revokeLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, revokeRule(w, r, db, dbAccount))
	}
}

func getLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbAccount))
	}
}

func addLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, createCrypto(w, r, db, dbAccount))
	}
}

func listLocAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbAccount))
	}
}

func deleteLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, deleteCrypto(w, r, db, dbAccount))
	}
}

func updateLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, updateCrypto(w, r, db, dbAccount))
	}
}

func replaceLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, replaceCrypto(w, r, db, dbAccount))
	}
}
