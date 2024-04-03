package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

func makeCredList(db database.ReadAccess, target model.CredOwnerTable) ([]string, error) {
	var creds model.Credentials
	if err := db.Select(&creds).Where(target.GetCredCond()).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	names := make([]string, len(creds))

	for i := range creds {
		names[i] = creds[i].Name
	}

	return names, nil
}

func getCred(r *http.Request, db database.ReadAccess, target model.CredOwnerTable,
) (*model.Credential, error) {
	name, ok := mux.Vars(r)["credential"]
	if !ok {
		return nil, notFound("missing credential name")
	}

	var cred model.Credential
	if err := db.Get(&cred, "name=?", name).And(target.GetCredCond()).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve credential %q: %w", name, err)
	}

	return &cred, nil
}

func addCredential(w http.ResponseWriter, r *http.Request, db *database.DB,
	target model.CredOwnerTable, protocol string,
) error {
	var jCred api.InCred
	if err := readJSON(r, &jCred); err != nil {
		return err
	}

	dbCred := &model.Credential{
		Name:   jCred.Name.Value,
		Type:   jCred.Type.Value,
		Value:  jCred.Value.Value,
		Value2: jCred.Value2.Value,
	}
	target.SetCredOwner(dbCred)

	if dbCred.Type == auth.PasswordHash {
		checkR66Password(dbCred, protocol)
	}

	if err := db.Insert(dbCred).Run(); err != nil {
		return fmt.Errorf("failed to insert credential: %w", err)
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Location", location(r.URL, dbCred.Name))

	return nil
}

func getCredential(w http.ResponseWriter, r *http.Request, db database.Access,
	target model.CredOwnerTable,
) error {
	dbCred, getErr := getCred(r, db, target)
	if getErr != nil {
		return getErr
	}

	w.WriteHeader(http.StatusOK)

	return writeJSON(w, api.OutCred{
		Name:   dbCred.Name,
		Type:   dbCred.Type,
		Value:  dbCred.Value,
		Value2: dbCred.Value2,
	})
}

func removeCredential(w http.ResponseWriter, r *http.Request, db database.Access,
	target model.CredOwnerTable,
) error {
	cred, getErr := getCred(r, db, target)
	if getErr != nil {
		return getErr
	}

	if err := db.Delete(cred).Run(); err != nil {
		return fmt.Errorf("failed to delete credential %q: %w", cred.Name, err)
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}
