package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl // duplicated code is about a different type
func getDBLocalAccount(r *http.Request, db *database.DB) (
	//nolint:unparam //keep returning the parent just in case we need it later
	*model.LocalAgent, *model.LocalAccount, error,
) {
	parent, parentErr := getDBServer(r, db)
	if parentErr != nil {
		return nil, nil, parentErr
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

		return parent, nil, fmt.Errorf("failed to retrieve local account %q: %w", login, err)
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

		if err := query.Where("local_agent_id=?", parent.ID).Run(); handleError(w, logger, err) {
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

//nolint:dupl //duplicated code is about a different type, best keep separate
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

		dbAccount := &model.LocalAccount{
			LocalAgentID: parent.ID,
			Login:        restAccount.Login.Value,
		}

		if tErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Insert(dbAccount).Run(); err != nil {
				return fmt.Errorf("failed to insert local account: %w", err)
			}

			if restAccount.Password.Valid {
				if err := updateAccountPassword(ses, dbAccount, restAccount.Password.Value); err != nil {
					return err
				}
			}

			return nil
		}); handleError(w, logger, tErr) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicated code is about a different type, best keep separate
func updateLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, oldAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restAccount := &api.InAccount{Login: api.AsNullable(oldAccount.Login)}
		if err := readJSON(r, restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := &model.LocalAccount{
			ID:           oldAccount.ID,
			LocalAgentID: oldAccount.LocalAgentID,
			Login:        restAccount.Login.Value,
		}

		if tErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Update(dbAccount).Run(); err != nil {
				return fmt.Errorf("failed to update local account: %w", err)
			}

			if restAccount.Password.Valid {
				if err := updateAccountPassword(ses, dbAccount, restAccount.Password.Value); err != nil {
					return err
				}
			}

			return nil
		}); handleError(w, logger, tErr) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for a different type, best keep separate
func replaceLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, oldAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := &model.LocalAccount{
			ID:           oldAccount.ID,
			LocalAgentID: oldAccount.LocalAgentID,
			Login:        restAccount.Login.Value,
		}

		if tErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Update(dbAccount).Run(); err != nil {
				return fmt.Errorf("failed to update local account: %w", err)
			}

			if err := updateAccountPassword(ses, dbAccount, restAccount.Password.Value); err != nil {
				return err
			}

			return nil
		}); handleError(w, logger, tErr) {
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

func addLocAccCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, addCredential(w, r, db, dbAccount))
	}
}

func getLocAccCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCredential(w, r, db, dbAccount))
	}
}

func removeLocAccCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getDBLocalAccount(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, removeCredential(w, r, db, acc))
	}
}

// Deprecated: replaced by Credentials.
func getLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func addLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, createCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func listLocAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func deleteLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, deleteCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func updateLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, updateCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func replaceLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBLocalAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, replaceCrypto(w, r, db, dbAccount))
	}
}
