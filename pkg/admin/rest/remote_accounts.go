package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func getRemAcc(r *http.Request, db *database.DB) (*model.RemoteAgent,
	*model.RemoteAccount, error) {
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
		if database.IsNotFound(err) {
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
		filters, err := parseListFilters(r, validSorting)
		if handleError(w, logger, err) {
			return
		}
		parent, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}
		filters.Conditions = builder.Eq{"remote_agent_id": parent.ID}

		var results []model.RemoteAccount
		if err := db.Select(&results, filters); handleError(w, logger, err) {
			return
		}

		ids := make([]uint64, len(results))
		for i, res := range results {
			ids[i] = res.ID
		}
		rules, err := getAuthorizedRuleList(db, typ, ids)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutAccount{"remoteAccounts": FromRemoteAccounts(results, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func getRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, result, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, FromRemoteAccount(result, rules))
		handleError(w, logger, err)
	}
}

func updateRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, old, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		account := newInRemAccount(old)
		if err := readJSON(r, account); handleError(w, logger, err) {
			return
		}

		if err := db.Update(accToRemote(account, parent, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(account.Login)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, old, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		var account api.InAccount
		if err := readJSON(r, &account); handleError(w, logger, err) {
			return
		}

		if err := db.Update(accToRemote(&account, parent, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(account.Login)))
		w.WriteHeader(http.StatusCreated)
	}
}

func addRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jsonAccount api.InAccount
		if err := readJSON(r, &jsonAccount); handleError(w, logger, err) {
			return
		}

		account := accToRemote(&jsonAccount, parent, 0)
		if err := db.Create(account); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, account.Login))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(acc); handleError(w, logger, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func authorizeRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = authorizeRule(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func revokeRemoteAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = revokeRule(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func getRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = getCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func addRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func listRemAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCertificates(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func deleteRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func updateRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func replaceRemAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getRemAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}
