package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// cryptoToModel transforms the JSON secure credentials into its database equivalent.
func cryptoToModel(a *api.InCrypto, c *model.Crypto) *model.Crypto {
	c.Name = str(a.Name)
	c.PrivateKey = types.CypherText(str(a.PrivateKey))
	c.SSHPublicKey = str(a.PublicKey)
	c.Certificate = str(a.Certificate)

	return c
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
func FromCryptos(dbCryptos []*model.Crypto) []*api.OutCrypto {
	outAuths := make([]*api.OutCrypto, len(dbCryptos))
	for i, dbCrypto := range dbCryptos {
		outAuths[i] = FromCrypto(dbCrypto)
	}

	return outAuths
}

func retrieveCrypto(r *http.Request, db *database.DB, owner model.CryptoOwner,
) (*model.Crypto, error) {
	cryptName, ok := mux.Vars(r)["certificate"]
	if !ok {
		return nil, notFound("missing certificate name")
	}

	var crypto model.Crypto
	if err := db.Get(&crypto, "name=?", cryptName).And(owner.GenCryptoSelectCond()).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("certificate '%s' not found", cryptName)
		}

		return nil, err
	}

	return &crypto, nil
}

func getCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CryptoOwner,
) error {
	result, err := retrieveCrypto(r, db, owner)
	if err != nil {
		return err
	}

	return writeJSON(w, FromCrypto(result))
}

func createCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CryptoOwner,
) error {
	var inCrypto api.InCrypto
	if err := readJSON(r, &inCrypto); err != nil {
		return err
	}

	dbCrypto := cryptoToModel(&inCrypto, &model.Crypto{})
	owner.SetCryptoOwner(dbCrypto)

	if err := db.Insert(dbCrypto).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", location(r.URL, dbCrypto.Name))
	w.WriteHeader(http.StatusCreated)

	warn := compatibility.CheckSHA1(dbCrypto.Certificate)
	if warn != "" {
		fmt.Fprint(w, warn)
	}

	return nil
}

func listCryptos(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CryptoOwner,
) error {
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

	query.Where(owner.GenCryptoSelectCond())

	if err := query.Run(); err != nil {
		return err
	}

	resp := map[string][]*api.OutCrypto{"certificates": FromCryptos(results)}

	return writeJSON(w, resp)
}

func deleteCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CryptoOwner,
) error {
	crypto, err := retrieveCrypto(r, db, owner)
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
	owner model.CryptoOwner,
) error {
	old, err := retrieveCrypto(r, db, owner)
	if err != nil {
		return err
	}

	var inAuth api.InCrypto
	if err := readJSON(r, &inAuth); err != nil {
		return err
	}

	crypto := cryptoToModel(&inAuth, old)
	if err := db.Update(crypto).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", locationUpdate(r.URL, crypto.Name))
	w.WriteHeader(http.StatusCreated)

	warn := compatibility.CheckSHA1(crypto.Certificate)
	if warn != "" {
		fmt.Fprint(w, warn)
	}

	return nil
}

func updateCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CryptoOwner,
) error {
	old, err := retrieveCrypto(r, db, owner)
	if err != nil {
		return err
	}

	inAuth := inCryptoFromModel(old)
	if err := readJSON(r, inAuth); err != nil {
		return err
	}

	crypto := cryptoToModel(inAuth, old)
	if err := db.Update(crypto).Run(); err != nil {
		return err
	}

	w.Header().Set("Location", locationUpdate(r.URL, crypto.Name))
	w.WriteHeader(http.StatusCreated)

	warn := compatibility.CheckSHA1(crypto.Certificate)
	if warn != "" {
		fmt.Fprint(w, warn)
	}

	return nil
}
