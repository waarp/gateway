package rest

import (
	"net/http"

	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InCert is the JSON representation of a certificate in requests made to
// the REST interface.
type InCert struct {
	Name        string `json:"name"`
	PrivateKey  []byte `json:"privateKey"`
	PublicKey   []byte `json:"publicKey"`
	Certificate []byte `json:"certificate"`
}

// ToModel transforms the JSON certificate into its database equivalent.
func (i *InCert) toModel(ownerType string, ownerID uint64) *model.Cert {
	return &model.Cert{
		OwnerType:   ownerType,
		OwnerID:     ownerID,
		Name:        i.Name,
		PrivateKey:  i.PrivateKey,
		PublicKey:   i.PublicKey,
		Certificate: i.Certificate,
	}
}

// OutCert is the JSON representation of a certificate in responses sent by
// the REST interface.
type OutCert struct {
	Name        string `json:"name"`
	PrivateKey  []byte `json:"privateKey"`
	PublicKey   []byte `json:"publicKey"`
	Certificate []byte `json:"certificate"`
}

// FromCert transforms the given database certificate into its JSON equivalent.
func FromCert(c *model.Cert) *OutCert {
	return &OutCert{
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
			Name:        cert.Name,
			PrivateKey:  cert.PrivateKey,
			PublicKey:   cert.PublicKey,
			Certificate: cert.Certificate,
		}
	}
	return certs
}

func getAgentInfo(r *http.Request, db *database.DB) (string, uint64, error) {
	var ownerType string
	var ownerID uint64
	if server, account, err := getLocAcc(r, db); err == nil {
		ownerType = account.TableName()
		ownerID = account.ID
	} else if server != nil {
		ownerType = server.TableName()
		ownerID = server.ID
	} else if partner, account, err := getRemAcc(r, db); err == nil {
		ownerType = account.TableName()
		ownerID = account.ID
	} else if partner != nil {
		ownerType = partner.TableName()
		ownerID = partner.ID
	} else {
		return "", 0, err
	}
	return ownerType, ownerID, nil
}

func getCert(r *http.Request, db *database.DB) (*model.Cert, error) {
	ownerType, ownerID, err := getAgentInfo(r, db)
	if err != nil {
		return nil, err
	}

	certName, ok := mux.Vars(r)["certificate"]
	if !ok {
		return nil, notFound("missing certificate name")
	}
	cert := &model.Cert{Name: certName, OwnerType: ownerType, OwnerID: ownerID}
	if err := db.Get(cert); err != nil {
		if err == database.ErrNotFound {
			return nil, notFound("certificate '%s' not found", certName)
		}
		return nil, err
	}
	return cert, nil
}

func getCertificate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getCert(r, db)
			if err != nil {
				return err
			}

			return writeJSON(w, FromCert(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createCertificate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ownerType, ownerID, err := getAgentInfo(r, db)
			if err != nil {
				return err
			}

			jsonCert := &InCert{}
			if err := readJSON(r, jsonCert); err != nil {
				return err
			}

			cert := jsonCert.toModel(ownerType, ownerID)
			if err := db.Create(cert); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, cert.Name))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listCertificates(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ownerType, ownerID, err := getAgentInfo(r, db)
			if err != nil {
				return err
			}

			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			filters.Conditions = builder.Eq{"owner_type": ownerType, "owner_id": ownerID}

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

func deleteCertificate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			cert, err := getCert(r, db)
			if err != nil {
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
func updateCertificate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getCert(r, db)
			if err != nil {
				return err
			}

			cert := &InCert{}
			if err := readJSON(r, cert); err != nil {
				return err
			}

			if err := db.Update(cert.toModel(check.OwnerType, check.OwnerID),
				check.ID, false); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r, cert.Name, check.Name))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
