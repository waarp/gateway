// Deprecated: cryptos have been replaced by the more generic auths.

package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// cryptoToModel transforms the JSON secure credentials into its database equivalent.
func cryptoToModel(in *api.InCrypto, a *model.Credential) *model.Credential {
	a.Name = in.Name.Value

	switch {
	case in.Certificate.Valid && in.PrivateKey.Valid:
		a.Type = auth.TLSCertificate
		a.Value = in.Certificate.Value
		a.Value2 = in.PrivateKey.Value
	case in.Certificate.Valid:
		a.Type = auth.TLSTrustedCertificate
		a.Value = in.Certificate.Value
	case in.PrivateKey.Valid:
		a.Type = sftp.AuthSSHPrivateKey
		a.Value = in.PrivateKey.Value
	case in.PublicKey.Valid:
		a.Type = sftp.AuthSSHPublicKey
		a.Value = in.PublicKey.Value
	}

	return a
}

func inCryptoFromModel(c *model.Credential) (*api.InCrypto, error) {
	switch c.Type {
	case auth.TLSTrustedCertificate:
		return &api.InCrypto{
			Name:        api.AsNullable(c.Name),
			Certificate: api.AsNullable(c.Value),
		}, nil
	case auth.TLSCertificate:
		return &api.InCrypto{
			Name:        api.AsNullable(c.Name),
			Certificate: api.AsNullable(c.Value),
			PrivateKey:  api.AsNullable(c.Value2),
		}, nil
	case sftp.AuthSSHPublicKey:
		return &api.InCrypto{
			Name:      api.AsNullable(c.Name),
			PublicKey: api.AsNullable(c.Value),
		}, nil
	case sftp.AuthSSHPrivateKey:
		return &api.InCrypto{
			Name:       api.AsNullable(c.Name),
			PrivateKey: api.AsNullable(c.Value),
		}, nil
	default:
		return nil, internal("unsupported certificate type '%s'", c.Type)
	}
}

// FromCrypto transforms the given database secure credentials into its JSON equivalent.
func FromCrypto(a *model.Credential) (*api.OutCrypto, error) {
	switch a.Type {
	case auth.TLSTrustedCertificate:
		return &api.OutCrypto{
			Name:        a.Name,
			Certificate: a.Value,
		}, nil
	case auth.TLSCertificate:
		return &api.OutCrypto{
			Name:        a.Name,
			Certificate: a.Value,
			PrivateKey:  a.Value2,
		}, nil
	case sftp.AuthSSHPublicKey:
		return &api.OutCrypto{
			Name:      a.Name,
			PublicKey: a.Value,
		}, nil
	case sftp.AuthSSHPrivateKey:
		return &api.OutCrypto{
			Name:       a.Name,
			PrivateKey: a.Value,
		}, nil
	default:
		return nil, internal("unsupported certificate type '%s'", a.Type)
	}
}

// FromCryptos transforms the given list of database secure credentials into its JSON
// equivalent.
func FromCryptos(dbCryptos []*model.Credential) ([]*api.OutCrypto, error) {
	outAuths := make([]*api.OutCrypto, len(dbCryptos))

	for i, dbCrypto := range dbCryptos {
		var err error
		if outAuths[i], err = FromCrypto(dbCrypto); err != nil {
			return nil, err
		}
	}

	return outAuths, nil
}

func retrieveCrypto(r *http.Request, db database.ReadAccess, owner model.CredOwnerTable) (*model.Credential, error) {
	cryptName, ok := mux.Vars(r)["certificate"]
	if !ok {
		return nil, notFound("missing certificate name")
	}

	var crypto model.Credential
	if err := db.Get(&crypto, "name=?", cryptName).And(owner.GetCredCond()).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("certificate '%s' not found", cryptName)
		}

		return nil, fmt.Errorf("failed to retrieve certificate %q: %w", cryptName, err)
	}

	return &crypto, nil
}

func getCrypto(w http.ResponseWriter, r *http.Request, db database.ReadAccess,
	owner model.CredOwnerTable,
) error {
	result, err := retrieveCrypto(r, db, owner)
	if err != nil {
		return err
	}

	crypto, err := FromCrypto(result)
	if err != nil {
		return err
	}

	return writeJSON(w, crypto)
}

func createCrypto(w http.ResponseWriter, r *http.Request, db database.Access,
	owner model.CredOwnerTable,
) error {
	var inCrypto api.InCrypto
	if err := readJSON(r, &inCrypto); err != nil {
		return err
	}

	dbCrypto := cryptoToModel(&inCrypto, &model.Credential{})
	owner.SetCredOwner(dbCrypto)

	if err := db.Insert(dbCrypto).Run(); err != nil {
		return fmt.Errorf("failed to insert certificate: %w", err)
	}

	w.Header().Set("Location", location(r.URL, dbCrypto.Name))
	w.WriteHeader(http.StatusCreated)

	if dbCrypto.Type == auth.TLSCertificate ||
		dbCrypto.Type == auth.TLSTrustedCertificate {
		warn := compatibility.CheckSHA1(dbCrypto.Value)
		if warn != "" {
			fmt.Fprint(w, warn)
		}
	}

	return nil
}

func listCryptos(w http.ResponseWriter, r *http.Request, db database.ReadAccess,
	owner model.CredOwnerTable,
) error {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	var results model.Credentials

	query, queryErr := parseSelectQuery(r, db, validSorting, &results)
	if queryErr != nil {
		return queryErr
	}

	query.Where(owner.GetCredCond())

	if err := query.Run(); err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	certs, convErr := FromCryptos(results)
	if convErr != nil {
		return convErr
	}

	resp := map[string][]*api.OutCrypto{"certificates": certs}

	return writeJSON(w, resp)
}

func deleteCrypto(w http.ResponseWriter, r *http.Request, db database.Access,
	owner model.CredOwnerTable,
) error {
	crypto, getErr := retrieveCrypto(r, db, owner)
	if getErr != nil {
		return getErr
	}

	if err := db.Delete(crypto).Run(); err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func replaceCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CredOwnerTable,
) error {
	old, getErr := retrieveCrypto(r, db, owner)
	if getErr != nil {
		return getErr
	}

	var inAuth api.InCrypto
	if jsonErr := readJSON(r, &inAuth); jsonErr != nil {
		return jsonErr
	}

	dbCrypto := cryptoToModel(&inAuth, old)

	if err := db.Update(dbCrypto).Run(); err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	w.Header().Set("Location", locationUpdate(r.URL, dbCrypto.Name))
	w.WriteHeader(http.StatusCreated)

	if dbCrypto.Type == auth.TLSCertificate ||
		dbCrypto.Type == auth.TLSTrustedCertificate {
		warn := compatibility.CheckSHA1(dbCrypto.Value)
		if warn != "" {
			fmt.Fprint(w, warn)
		}
	}

	return nil
}

func updateCrypto(w http.ResponseWriter, r *http.Request, db *database.DB,
	owner model.CredOwnerTable,
) error {
	old, getErr := retrieveCrypto(r, db, owner)
	if getErr != nil {
		return getErr
	}

	inAuth, convErr := inCryptoFromModel(old)
	if convErr != nil {
		return convErr
	}

	if jsonErr := readJSON(r, inAuth); jsonErr != nil {
		return jsonErr
	}

	dbCrypto := cryptoToModel(inAuth, old)

	if err := db.Update(dbCrypto).Run(); err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	w.Header().Set("Location", locationUpdate(r.URL, dbCrypto.Name))
	w.WriteHeader(http.StatusCreated)

	if dbCrypto.Type == auth.TLSCertificate ||
		dbCrypto.Type == auth.TLSTrustedCertificate {
		warn := compatibility.CheckSHA1(dbCrypto.Value)
		if warn != "" {
			fmt.Fprint(w, warn)
		}
	}

	return nil
}
