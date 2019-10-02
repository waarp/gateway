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

func createCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cert := &model.CertChain{}

		if err := restCreate(db, r, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}

		id := strconv.FormatUint(cert.ID, 10)
		w.Header().Set("Location", RestURI+CertsURI+"/"+id)
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

		conditions := make([]builder.Cond, 0)
		accounts := r.Form["account"]
		partners := r.Form["parnter"]
		if len(accounts) > 0 {
			ids := make([]uint64, len(accounts))
			for i, account := range accounts {
				id, err := strconv.ParseUint(account, 10, 64)
				if err != nil {
					handleErrors(w, logger, &badRequest{
						msg: fmt.Sprintf("'%s' is not a valid account ID", account)})
					return
				}
				ids[i] = id
			}

			conditions = append(conditions, builder.Eq{"owner_type": "ACCOUNT"})
			conditions = append(conditions, builder.In("owner_id", ids))
		} else if len(partners) > 0 {
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

			conditions = append(conditions, builder.Eq{"owner_type": "PARTNER"})
			conditions = append(conditions, builder.In("owner_id", ids))
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
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
		id, err := strconv.ParseUint(mux.Vars(r)["certificate"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		cert := &model.CertChain{ID: id}

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
		id, err := strconv.ParseUint(mux.Vars(r)["certificate"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		cert := &model.CertChain{ID: id}
		if err := restDelete(db, cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(mux.Vars(r)["certificate"], 10, 64)
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		oldCert := &model.CertChain{ID: id}
		newCert := &model.CertChain{ID: id}

		if err := restUpdate(db, r, oldCert, newCert); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", RestURI+CertsURI+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
