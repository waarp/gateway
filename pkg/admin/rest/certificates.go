package rest

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InCert is the JSON representation of a certificate in requests made to
// the REST interface.
type InCert struct {
	OwnerType   string `json:"ownerType"`
	OwnerID     uint64 `json:"ownerID"`
	Name        string `json:"name"`
	PrivateKey  []byte `json:"privateKey"`
	PublicKey   []byte `json:"publicKey"`
	Certificate []byte `json:"certificate"`
}

// ToModel transforms the JSON certificate into its database equivalent.
func (i *InCert) toModel() *model.Cert {
	return &model.Cert{
		OwnerType:   i.OwnerType,
		OwnerID:     i.OwnerID,
		Name:        i.Name,
		PrivateKey:  i.PrivateKey,
		PublicKey:   i.PublicKey,
		Certificate: i.Certificate,
	}
}

// OutCert is the JSON representation of a certificate in responses sent by
// the REST interface.
type OutCert struct {
	ID          uint64 `json:"id"`
	OwnerType   string `json:"ownerType"`
	OwnerID     uint64 `json:"ownerID"`
	Name        string `json:"name"`
	PrivateKey  []byte `json:"privateKey"`
	PublicKey   []byte `json:"publicKey"`
	Certificate []byte `json:"certificate"`
}

// FromCert transforms the given database certificate into its JSON equivalent.
func fromCert(c *model.Cert) *OutCert {
	return &OutCert{
		ID:          c.ID,
		OwnerType:   c.OwnerType,
		OwnerID:     c.OwnerID,
		Name:        c.Name,
		PrivateKey:  c.PrivateKey,
		PublicKey:   c.PublicKey,
		Certificate: c.Certificate,
	}
}

// FromCerts transforms the given list of database certificates into its JSON
// equivalent.
func FromCerts(cs []model.Cert) []OutCert {
	certs := make([]OutCert, len(cs))
	for i, cert := range cs {
		certs[i] = OutCert{
			ID:          cert.ID,
			OwnerType:   cert.OwnerType,
			OwnerID:     cert.OwnerID,
			Name:        cert.Name,
			PrivateKey:  cert.PrivateKey,
			PublicKey:   cert.PublicKey,
			Certificate: cert.Certificate,
		}
	}
	return certs
}

func parseOwnerParam(r *http.Request, filters *database.Filters) error {
	ownerTypes := []string{"local_agents", "remote_agents", "local_accounts", "remote_accounts"}
	conditions := []builder.Cond{}
	for _, ownerType := range ownerTypes {
		owners := r.Form[ownerType]

		if len(owners) > 0 {
			ownerIDs := make([]uint64, len(owners))
			for i, owner := range owners {
				id, err := strconv.ParseUint(owner, 10, 64)
				if err != nil {
					return &badRequest{msg: fmt.Sprintf("'%s' is not a valid %s ID",
						owner, ownerType)}
				}
				ownerIDs[i] = id
			}
			condition := builder.And(builder.Eq{"owner_type": ownerType},
				builder.In("owner_id", ownerIDs))
			conditions = append(conditions, condition)
		}
	}
	filters.Conditions = builder.Or(conditions...)
	return nil
}

func getCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "certificate")
			if err != nil {
				return err
			}
			result := &model.Cert{ID: id}

			if err := get(db, result); err != nil {
				return err
			}

			return writeJSON(w, fromCert(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonCert := &InCert{}
			if err := readJSON(r, jsonCert); err != nil {
				return err
			}

			cert := jsonCert.toModel()
			if err := db.Create(cert); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, cert.ID))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listCertificates(logger *log.Logger, db *database.Db) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			if err := parseOwnerParam(r, filters); err != nil {
				return err
			}

			var results []model.Cert
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutCert{"certificates": FromCerts(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "certificate")
			if err != nil {
				return &notFound{}
			}

			cert := &model.Cert{ID: id}
			if err := get(db, cert); err != nil {
				return err
			}

			if err := db.Delete(cert); err != nil {
				return err
			}
			w.WriteHeader(http.StatusNoContent)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func updateCertificate(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "certificate")
			if err != nil {
				return &notFound{}
			}

			if err := exist(db, &model.Cert{ID: id}); err != nil {
				return err
			}

			cert := &InCert{}
			if err := readJSON(r, cert); err != nil {
				return err
			}

			if err := db.Update(cert.toModel(), id, false); err != nil {
				return err
			}

			w.Header().Set("Location", location(r))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
