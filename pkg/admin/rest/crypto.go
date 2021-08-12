package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// cryptoToModel transforms the JSON secure credentials into its database equivalent.
func cryptoToModel(a *api.InCrypto, id uint64, ownerType string, ownerID uint64) *model.Crypto {
	return &model.Crypto{
		ID:           id,
		OwnerType:    ownerType,
		OwnerID:      ownerID,
		Name:         str(a.Name),
		PrivateKey:   types.CypherText(str(a.PrivateKey)),
		SSHPublicKey: str(a.PublicKey),
		Certificate:  str(a.Certificate),
	}
}

func inCryptoFromModel(c *model.Crypto) *api.InCrypto {
	return &api.InCrypto{
		Name:        &c.Name,
		PrivateKey:  strPtr(string(c.PrivateKey)),
		PublicKey:   &c.SSHPublicKey,
		Certificate: &c.Certificate,
	}
}

// FromCrypto transforms the given database secure credentials into its JSON equivalent.
func FromCrypto(a *model.Crypto) *api.OutCrypto {
	return &api.OutCrypto{
		Name:        a.Name,
		PrivateKey:  string(a.PrivateKey),
		PublicKey:   a.SSHPublicKey,
		Certificate: a.Certificate,
	}
}

// FromCryptos transforms the given list of database secure credentials into its JSON
// equivalent.
func FromCryptos(cryptos []model.Crypto) []api.OutCrypto {
	outAuths := make([]api.OutCrypto, len(cryptos))
	for i := range cryptos {
		outAuths[i] = *FromCrypto(&cryptos[i])
	}
	return outAuths
}

func retrieveCrypto(r *http.Request, db *database.DB, ownerType string, ownerID uint64) (*model.Crypto, error) {
	cryptName, ok := mux.Vars(r)["certificate"]
	if !ok {
		return nil, notFound("missing certificate name")
	}
	var crypto model.Crypto
	if err := db.Get(&crypto, "name=? AND owner_type=? AND owner_id=?", cryptName,
		ownerType, ownerID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("certificate '%s' not found", cryptName)
		}
		return nil, err
	}
	return &crypto, nil
}

func getCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	result, err := retrieveCrypto(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	return writeJSON(w, FromCrypto(result))
}

func createCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	var inCrypto api.InCrypto
	if err := readJSON(r, &inCrypto); err != nil {
		return err
	}

	crypto := cryptoToModel(&inCrypto, 0, ownerType, ownerID)
	if err := db.Insert(crypto).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", location(r.URL, crypto.Name))
	w.WriteHeader(http.StatusCreated)
	return nil
}

func listCryptos(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	var results model.Cryptos
	query, err := parseSelectQuery(r, db, validSorting, &results)
	if err != nil {
		return err
	}

	query.Where("owner_type=? AND owner_id=?", ownerType, ownerID)

	if err := query.Run(); err != nil {
		return err
	}

	resp := map[string][]api.OutCrypto{"certificates": FromCryptos(results)}
	return writeJSON(w, resp)
}

func deleteCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	crypto, err := retrieveCrypto(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	if err := db.Delete(crypto).Run(); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func replaceCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	old, err := retrieveCrypto(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	var inAuth api.InCrypto
	if err := readJSON(r, &inAuth); err != nil {
		return err
	}

	crypto := cryptoToModel(&inAuth, old.ID, ownerType, ownerID)
	if err := db.Update(crypto).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", locationUpdate(r.URL, crypto.Name))
	w.WriteHeader(http.StatusCreated)
	return nil
}

func updateCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	ownerType string, ownerID uint64) error {

	old, err := retrieveCrypto(r, db, ownerType, ownerID)
	if err != nil {
		return err
	}

	inAuth := inCryptoFromModel(old)
	if err := readJSON(r, inAuth); err != nil {
		return err
	}

	crypto := cryptoToModel(inAuth, old.ID, ownerType, ownerID)
	if err := db.Update(crypto).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", locationUpdate(r.URL, crypto.Name))
	w.WriteHeader(http.StatusCreated)
	return nil
}
