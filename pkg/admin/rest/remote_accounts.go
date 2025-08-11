package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:dupl // duplicated code is about a different type
func getDBRemoteAccount(r *http.Request, db *database.DB) (*model.RemoteAgent,
	*model.RemoteAccount, error,
) {
	parent, parentErr := retrievePartner(r, db)
	if parentErr != nil {
		return nil, nil, parentErr
	}

	login, ok := mux.Vars(r)["remote_account"]
	if !ok {
		return parent, nil, notFound("missing partner name")
	}

	var dbAccount model.RemoteAccount
	if err := db.Get(&dbAccount, "login=? AND remote_agent_id=?", login, parent.ID).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return parent, nil, notFoundf("no account %q found for partner %q",
				login, parent.Name)
		}

		return parent, nil, fmt.Errorf("failed to retrieve remote account %q: %w", login, err)
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

		response := map[string][]*api.OutRemoteAccount{"remoteAccounts": restAccounts}
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

//nolint:dupl //duplicated code is about a different type, best keep separate
func replaceRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, oldAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InRemoteAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := &model.RemoteAccount{
			ID:            oldAccount.ID,
			RemoteAgentID: parent.ID,
			Login:         restAccount.Login.Value,
		}

		if tErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Update(dbAccount).Run(); err != nil {
				return fmt.Errorf("failed to update remote account: %w", err)
			}

			return updateAccountPassword(ses, dbAccount, restAccount.Password.Value)
		}); handleError(w, logger, tErr) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAccount.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicated code is about a different type, best keep separate
func updateRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, oldAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restAccount := &api.InRemoteAccount{Login: asNullable(oldAccount.Login)}
		if err := readJSON(r, restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := &model.RemoteAccount{
			ID:            oldAccount.ID,
			RemoteAgentID: parent.ID,
			Login:         restAccount.Login.Value,
		}

		if tErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Update(dbAccount).Run(); err != nil {
				return fmt.Errorf("failed to update remote account: %w", err)
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
func addRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, getErr := retrievePartner(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restAccount api.InRemoteAccount
		if err := readJSON(r, &restAccount); handleError(w, logger, err) {
			return
		}

		dbAccount := &model.RemoteAccount{
			RemoteAgentID: parent.ID,
			Login:         restAccount.Login.Value,
		}

		if tErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Insert(dbAccount).Run(); err != nil {
				return fmt.Errorf("failed to insert remote account: %w", err)
			}

			if restAccount.Password.Value != "" {
				if err := ses.Insert(&model.Credential{
					RemoteAccountID: utils.NewNullInt64(dbAccount.ID),
					Type:            auth.Password,
					Value:           restAccount.Password.Value,
				}).Run(); err != nil {
					return fmt.Errorf("failed to insert password credential: %w", err)
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

func deleteRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
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

func addRemAccCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, addCredential(w, r, db, dbAccount))
	}
}

func getRemAccCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCredential(w, r, db, dbAccount))
	}
}

func removeRemAccCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, removeCredential(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func getRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func addRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, createCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func listRemAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func deleteRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, deleteCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func updateRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, updateCrypto(w, r, db, dbAccount))
	}
}

// Deprecated: replaced by Credentials.
func replaceRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, dbAccount, getErr := getDBRemoteAccount(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, replaceCrypto(w, r, db, dbAccount))
	}
}
