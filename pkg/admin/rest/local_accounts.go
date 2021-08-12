package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getLocAcc(r *http.Request, db *database.DB) (*model.LocalAgent, *model.LocalAccount, error) {
	parent, err := getServ(r, db)
	if err != nil {
		return nil, nil, err
	}

	login, ok := mux.Vars(r)["local_account"]
	if !ok {
		return parent, nil, notFound("missing account login")
	}

	var account model.LocalAccount
	if err := db.Get(&account, "login=? AND local_agent_id=?", login, parent.ID).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return parent, nil, notFound("no account '%s' found for server %s",
				login, parent.Name)
		}
		return parent, nil, err
	}
	return parent, &account, nil
}

func getLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, result, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, FromLocalAccount(result, rules))
		handleError(w, logger, err)
	}
}

//nolint:dupl
func listLocalAccounts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"login", true},
		"login+":  order{"login", true},
		"login-":  order{"login", false},
	}
	typ := (&model.LocalAccount{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		var accounts model.LocalAccounts
		query, err := parseSelectQuery(r, db, validSorting, &accounts)
		if handleError(w, logger, err) {
			return
		}
		parent, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}
		query.Where("local_agent_id=?", parent.ID)

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		ids := make([]uint64, len(accounts))
		for i := range accounts {
			ids[i] = accounts[i].ID
		}
		rules, err := getAuthorizedRuleList(db, typ, ids)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutAccount{"localAccounts": FromLocalAccounts(accounts, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func addLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, err := getServ(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jAcc api.InAccount
		if err := readJSON(r, &jAcc); handleError(w, logger, err) {
			return
		}

		account, err := accToLocal(&jAcc, parent, 0)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(account).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, account.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, old, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		jAcc := newInLocAccount(old)
		if err := readJSON(r, jAcc); handleError(w, logger, err) {
			return
		}

		acc, err := accToLocal(jAcc, parent, old.ID)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Update(acc).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, acc.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, old, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jAcc api.InAccount
		if err := readJSON(r, &jAcc); handleError(w, logger, err) {
			return
		}

		acc, err := accToLocal(&jAcc, parent, old.ID)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Update(acc).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, acc.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(acc).Run(); handleError(w, logger, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = authorizeRule(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func revokeLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = revokeRule(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func getLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = getCrypto(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func addLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCrypto(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func listLocAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCryptos(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func deleteLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCrypto(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func updateLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCrypto(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func replaceLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCrypto(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}
