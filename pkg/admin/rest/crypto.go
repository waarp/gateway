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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// cryptoToModel transforms the JSON secure credentials into its database equivalent.
func cryptoToModel(in *api.InCrypto, cred *model.Credential) *model.Credential {
	cred.Name = in.Name.Value

	switch {
	case in.Certificate.Valid && in.PrivateKey.Valid:
		if compatibility.IsLegacyR66CertPEM(in.Certificate.Value) {
			cred.Type = r66.AuthLegacyCertificate
		} else {
			cred.Type = auth.TLSCertificate
			cred.Value = in.Certificate.Value
			cred.Value2 = in.PrivateKey.Value
		}
	case in.Certificate.Valid:
		if compatibility.IsLegacyR66CertPEM(in.Certificate.Value) {
			cred.Type = r66.AuthLegacyCertificate
		} else {
			cred.Type = auth.TLSTrustedCertificate
			cred.Value = in.Certificate.Value
		}
	case in.PrivateKey.Valid:
		cred.Type = sftp.AuthSSHPrivateKey
		cred.Value = in.PrivateKey.Value
	case in.PublicKey.Valid:
		cred.Type = sftp.AuthSSHPublicKey
		cred.Value = in.PublicKey.Value
	}

	return cred
}

func inCryptoFromModel(c *model.Credential) (*api.InCrypto, error) {
	switch c.Type {
	case auth.TLSTrustedCertificate:
		return &api.InCrypto{
			Name:        asNullableStr(c.Name),
			Certificate: asNullableStr(c.Value),
		}, nil
	case auth.TLSCertificate:
		return &api.InCrypto{
			Name:        asNullableStr(c.Name),
			Certificate: asNullableStr(c.Value),
			PrivateKey:  asNullableStr(c.Value2),
		}, nil
	case sftp.AuthSSHPublicKey:
		return &api.InCrypto{
			Name:      asNullableStr(c.Name),
			PublicKey: asNullableStr(c.Value),
		}, nil
	case sftp.AuthSSHPrivateKey:
		return &api.InCrypto{
			Name:       asNullableStr(c.Name),
			PrivateKey: asNullableStr(c.Value),
		}, nil
	case r66.AuthLegacyCertificate:
		if c.LocalAgentID.Valid || c.RemoteAccountID.Valid {
			return &api.InCrypto{
				Name:        asNullableStr(c.Name),
				Certificate: asNullableStr(compatibility.LegacyR66CertPEM),
				PrivateKey:  asNullableStr(compatibility.LegacyR66KeyPEM),
			}, nil
		} else {
			return &api.InCrypto{
				Name:        asNullableStr(c.Name),
				Certificate: asNullableStr(compatibility.LegacyR66CertPEM),
			}, nil
		}
	default:
		return nil, internal("unsupported certificate type '%s'", c.Type)
	}
}

// FromCrypto transforms the given database secure credentials into its JSON equivalent.
func FromCrypto(cred *model.Credential) (*api.OutCrypto, error) {
	switch cred.Type {
	case auth.TLSTrustedCertificate:
		return &api.OutCrypto{
			Name:        cred.Name,
			Certificate: cred.Value,
		}, nil
	case auth.TLSCertificate:
		return &api.OutCrypto{
			Name:        cred.Name,
			Certificate: cred.Value,
			PrivateKey:  cred.Value2,
		}, nil
	case sftp.AuthSSHPublicKey:
		return &api.OutCrypto{
			Name:      cred.Name,
			PublicKey: cred.Value,
		}, nil
	case sftp.AuthSSHPrivateKey:
		return &api.OutCrypto{
			Name:       cred.Name,
			PrivateKey: cred.Value,
		}, nil
	case r66.AuthLegacyCertificate:
		if cred.LocalAgentID.Valid || cred.RemoteAccountID.Valid {
			return &api.OutCrypto{
				Name:        cred.Name,
				Certificate: compatibility.LegacyR66CertPEM,
				PrivateKey:  compatibility.LegacyR66KeyPEM,
			}, nil
		} else {
			return &api.OutCrypto{
				Name:        cred.Name,
				Certificate: compatibility.LegacyR66CertPEM,
			}, nil
		}
	default:
		return nil, internal("unsupported certificate type '%s'", cred.Type)
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
