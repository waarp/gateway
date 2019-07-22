package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func getPartnerID(r *http.Request, db *database.Db) (uint64, error) {
	partner := &model.Partner{
		Name: mux.Vars(r)["partner"],
	}

	if err := db.Get(partner); err != nil {
		if err == database.ErrNotFound {
			return 0, &notFound{}
		}
		return 0, err
	}

	return partner.ID, nil
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

		id, err := getPartnerID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		cond := builder.In("partner_id", id)

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: cond,
		}

		results := make([]model.Account, 0)
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}
		for i := range results {
			results[i].Password = nil
		}

		resp := map[string][]model.Account{"accounts": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getPartnerID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		account := &model.Account{}

		if err := readJSON(r, account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		account.PartnerID = id

		test := &model.Account{
			PartnerID: account.PartnerID,
			Username:  account.Username,
		}

		if err := restCreate(db, account, test); err != nil {
			handleErrors(w, logger, err)
			return
		}
		partnerName := mux.Vars(r)["partner"]
		w.Header().Set("Location", RestURI+PartnersURI+partnerName+AccountsURI+"/"+account.Username)
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getPartnerID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		account := &model.Account{
			PartnerID: id,
			Username:  mux.Vars(r)["account"],
		}
		if err := restDelete(db, account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getPartnerID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		old := &model.Account{
			PartnerID: id,
			Username:  mux.Vars(r)["account"],
		}

		account := &model.Account{}
		if r.Method == http.MethodPatch {
			if err := restGet(db, account); err != nil {
				handleErrors(w, logger, err)
				return
			}
		}

		if err := readJSON(r, account); err != nil {
			handleErrors(w, logger, err)
			return
		}

		if err := restUpdate(db, old, account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		partnerName := mux.Vars(r)["partner"]
		w.Header().Set("Location", RestURI+PartnersURI+partnerName+AccountsURI+"/"+account.Username)
		w.WriteHeader(http.StatusCreated)
	}
}
