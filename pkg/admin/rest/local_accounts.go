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

func getLocAcc(r *http.Request, db *database.DB) (*model.LocalAgent, *model.LocalAccount, error) {
	parent, err := getLocAg(r, db)
	if err != nil {
		return nil, nil, err
	}

	login, ok := mux.Vars(r)["local_account"]
	if !ok {
		return parent, nil, notFound("missing account login")
	}

	result := &model.LocalAccount{}
	result.LocalAgentID = parent.ID
	result.Login = login

	if err := db.Get(result); err != nil {
		if database.IsNotFound(err) {
			return parent, nil, notFound("no account '%s' found for server %s",
				login, parent.Name)
		}
		return parent, nil, err
	}
	return parent, result, nil
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
	validSorting := map[string]string{
		"default": "login ASC",
		"login+":  "login ASC",
		"login-":  "login DESC",
	}
	typ := (&model.LocalAccount{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		filters, err := parseListFilters(r, validSorting)
		if handleError(w, logger, err) {
			return
		}
		parent, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}
		filters.Conditions = builder.Eq{"local_agent_id": parent.ID}

		var results []model.LocalAccount
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

		resp := map[string][]api.OutAccount{"localAccounts": FromLocalAccounts(results, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func addLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, err := getLocAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		var acc api.InAccount
		if err := readJSON(r, &acc); handleError(w, logger, err) {
			return
		}

		account := accToLocal(&acc, parent, 0)
		if err := db.Create(account); handleError(w, logger, err) {
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

		acc := newInLocAccount(old)
		if err := readJSON(r, acc); handleError(w, logger, err) {
			return
		}

		if err := db.Update(accToLocal(acc, parent, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(acc.Login)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parent, old, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		var acc api.InAccount
		if err := readJSON(r, &acc); handleError(w, logger, err) {
			return
		}

		if err := db.Update(accToLocal(&acc, parent, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(acc.Login)))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteLocalAccount(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(acc); handleError(w, logger, err) {
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

		err = getCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func addLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func listLocAccountCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCertificates(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func deleteLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func updateLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}

func replaceLocAccountCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, acc, err := getLocAcc(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCertificate(w, r, db, acc.TableName(), acc.ID)
		handleError(w, logger, err)
	}
}
