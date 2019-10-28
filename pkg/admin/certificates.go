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
		cert := model.Cert{}

		if err := restCreate(db, r, &cert); err != nil {
			handleErrors(w, logger, err)
			return
		}

		id := strconv.FormatUint(cert.ID, 10)
		w.Header().Set("Location", APIPath+CertificatesPath+"/"+id)
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

		validOwners := []string{"local_agents", "remote_agents", "local_accounts",
			"remote_accounts"}
		conditions := make([]builder.Cond, 0)

		for _, ownerType := range validOwners {
			owners := r.Form[ownerType]

			if len(owners) > 0 {
				ownerIDs := make([]uint64, len(owners))
				for i, owner := range owners {
					id, err := strconv.ParseUint(owner, 10, 64)
					if err != nil {
						msg := fmt.Sprintf("'%s' is not a valid %s ID", owner, ownerType)
						handleErrors(w, logger, &badRequest{msg})
						return
					}
					ownerIDs[i] = id
				}
				condition := builder.And(builder.Eq{"owner_type": ownerType},
					builder.In("owner_id", ownerIDs))
				conditions = append(conditions, condition)
			}
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.Or(conditions...),
		}

		results := []model.Cert{}
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.Cert{"certificates": results}
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
		cert := model.Cert{ID: id}

		if err := restGet(db, &cert); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, &cert); err != nil {
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
		cert := model.Cert{ID: id}

		if err := restDelete(db, &cert); err != nil {
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
		newCert := model.Cert{}

		if err := restUpdate(db, r, &newCert, id); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", APIPath+CertificatesPath+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
