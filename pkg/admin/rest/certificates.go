package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InCert is the JSON representation of a certificate in requests made to
// the REST interface.
type InCert struct {
	Name        *string `json:"name,omitempty"`
	PrivateKey  []byte  `json:"privateKey,omitempty"`
	PublicKey   []byte  `json:"publicKey,omitempty"`
	Certificate []byte  `json:"certificate,omitempty"`
}

// ruleUptToDB transforms the JSON certificate into its database equivalent.
func (i *InCert) toModel(id uint64, ownerType string, ownerID uint64) *model.Cert {
	return &model.Cert{
		ID:          id,
		OwnerType:   ownerType,
		OwnerID:     ownerID,
		Name:        str(i.Name),
		PrivateKey:  i.PrivateKey,
		PublicKey:   i.PublicKey,
		Certificate: i.Certificate,
	}
}

func inCertFromModel(c *model.Cert) *InCert {
	return &InCert{
		Name:        &c.Name,
		PrivateKey:  c.PrivateKey,
		PublicKey:   c.Certificate,
		Certificate: c.Certificate,
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

func getCert(r *http.Request, db *database.DB, ownerType string, ownerID uint64) (*model.Cert, error) {
	certName, ok := mux.Vars(r)["certificate"]
	if !ok {
		return nil, notFound("missing certificate name")
	}
	var cert model.Cert
	if err := db.Get(&cert, "name=? AND owner_type=? AND owner_id=?", certName,
		ownerType, ownerID).Run(); err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			return nil, notFound("certificate '%s' not found", certName)
		}
		return nil, err
	}
	return &cert, nil
}

func getCertificate(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	result, err := getCert(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	return writeJSON(w, FromCert(result))
}

func createCertificate(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	var jsonCert InCert
	if err := readJSON(r, &jsonCert); err != nil {
		return err
	}

	cert := jsonCert.toModel(0, ownerType, ownerID)
	if err := db.Insert(cert).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", location(r.URL, cert.Name))
	w.WriteHeader(http.StatusCreated)
	return nil
}

func listCertificates(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	var results model.Certificates
	query, err := parseSelectQuery(r, db, validSorting, &results)
	if err != nil {
		return err
	}

	query.Where("owner_type=? AND owner_id=?", ownerType, ownerID)

	if err := query.Run(); err != nil {
		return err
	}

	resp := map[string][]OutCert{"certificates": FromCerts(results)}
	return writeJSON(w, resp)
}

func deleteCertificate(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	cert, err := getCert(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	if err := db.Delete(cert).Run(); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func replaceCertificate(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	old, err := getCert(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	var jCert InCert
	if err := readJSON(r, &jCert); err != nil {
		return err
	}

	cert := jCert.toModel(old.ID, ownerType, ownerID)
	if err := db.Update(cert).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", locationUpdate(r.URL, cert.Name))
	w.WriteHeader(http.StatusCreated)
	return nil
}

func updateCertificate(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	old, err := getCert(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	jCert := inCertFromModel(old)
	if err := readJSON(r, jCert); err != nil {
		return err
	}

	cert := jCert.toModel(old.ID, ownerType, ownerID)
	if err := db.Update(cert).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", locationUpdate(r.URL, cert.Name))
	w.WriteHeader(http.StatusCreated)
	return nil
}
