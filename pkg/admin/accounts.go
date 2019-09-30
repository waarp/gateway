package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["account"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := &model.Account{ID: id}

		if err := restGet(db, account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, account); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func listAccounts(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "username"
		validSorting := []string{"username"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		partners := r.Form["partner"]
		conditions := make([]builder.Cond, 0)
		if len(partners) > 0 {
			ids := make([]uint64, len(partners))
			for i, partner := range partners {
				id, err := strconv.ParseUint(partner, 10, 64)
				if err != nil {
					handleErrors(w, logger, &badRequest{
						msg: fmt.Sprintf("'%s' is not a valid partner ID", partner)})
					return
				}
				ids[i] = id
			}

			conditions = append(conditions, builder.In("partner_id", ids))
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
		}

		results := make([]model.Account, 0)
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.Account{"accounts": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := &model.Account{}

		if err := restCreate(db, r, account); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(account.ID, 10)
		w.Header().Set("Location", RestURI+AccountsURI+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["account"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := &model.Account{ID: id}

		if err := restDelete(db, account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["account"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		oldAcc := &model.Account{ID: id}
		newAcc := &model.Account{ID: id}

		if err := restUpdate(db, r, oldAcc, newAcc); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", RestURI+AccountsURI+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
