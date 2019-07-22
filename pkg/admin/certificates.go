package admin

import (
	"net/http"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getAccountID(r *http.Request, db *database.Db) (uint64, error) {
	partnerID, err := getPartnerID(r, db)
	if err != nil {
		return 0, err
	}

	account := &model.Account{
		Username:  mux.Vars(r)["account"],
		PartnerID: partnerID,
	}

	if err := db.Get(account); err != nil {
		if err == database.ErrNotFound {
			return 0, &notFound{}
		}
		return 0, err
	}

	return account.ID, nil
}

func createCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getAccountID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		cert := &model.CertChain{}

		if err := readJSON(r, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		cert.AccountID = id

		test := &model.CertChain{
			AccountID: cert.AccountID,
			Name:      cert.Name,
		}

		if err := restCreate(db, cert, test); err != nil {
			handleErrors(w, logger, err)
			return
		}
		partnerName := mux.Vars(r)["partner"]
		accountName := mux.Vars(r)["account"]
		w.Header().Set("Location", RestURI+PartnersURI+partnerName+AccountsURI+
			"/"+accountName+CertsURI+"/"+cert.Name)
		w.WriteHeader(http.StatusCreated)
	}
}

func listCertificates(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "name"
		validSorting := []string{"name"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		id, err := getAccountID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		cond := builder.In("account_id", id)

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: cond,
		}

		results := &[]*model.CertChain{}
		if err := db.Select(results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string]*[]*model.CertChain{"certificates": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getAccountID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		cert := &model.CertChain{
			Name:      mux.Vars(r)["certificate"],
			AccountID: id,
		}

		if err := restGet(db, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func deleteCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getAccountID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		cert := &model.CertChain{
			Name:      mux.Vars(r)["certificate"],
			AccountID: id,
		}
		if err := restDelete(db, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getAccountID(r, db)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		old := &model.CertChain{
			Name:      mux.Vars(r)["certificate"],
			AccountID: id,
		}

		cert := &model.CertChain{}
		if r.Method == http.MethodPatch {
			if err := restGet(db, cert); err != nil {
				handleErrors(w, logger, err)
				return
			}
		}

		if err := readJSON(r, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}

		if err := restUpdate(db, old, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		partnerName := mux.Vars(r)["partner"]
		accountName := mux.Vars(r)["account"]
		w.Header().Set("Location", RestURI+PartnersURI+partnerName+AccountsURI+
			"/"+accountName+CertsURI+"/"+cert.Name)
		w.WriteHeader(http.StatusCreated)
	}
}
