package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func retrieveDBPGPKey(r *http.Request, db database.ReadAccess) (*model.PGPKey, error) {
	keyName, ok := mux.Vars(r)["pgp_key"]
	if !ok {
		return nil, notFound("missing PGP key name")
	}

	var dbKey model.PGPKey
	if err := db.Get(&dbKey, "name=?", keyName).Run(); database.IsNotFound(err) {
		return nil, notFound("PGP key %q not found", keyName)
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	return &dbKey, nil
}

func dbPGPKeyToREST(dbKey *model.PGPKey) *api.GetPGPKeyRespObject {
	return &api.GetPGPKeyRespObject{
		Name:       dbKey.Name,
		PublicKey:  dbKey.PublicKey,
		PrivateKey: dbKey.PrivateKey.String(),
	}
}

func dbPGPKeysToREST(dbKeys []*model.PGPKey) []*api.GetPGPKeyRespObject {
	restKeys := make([]*api.GetPGPKeyRespObject, len(dbKeys))
	for i := range dbKeys {
		restKeys[i] = dbPGPKeyToREST(dbKeys[i])
	}

	return restKeys
}

func addPGPKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restKey api.PostPGPKeyReqObject
		if err := readJSON(r, &restKey); handleError(w, logger, err) {
			return
		}

		dbKey := &model.PGPKey{
			Name:       restKey.Name,
			PrivateKey: database.SecretText(restKey.PrivateKey),
			PublicKey:  restKey.PublicKey,
		}
		if err := db.Insert(dbKey).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbKey.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for a different type, keep separate
func listPGPKeys(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbKeys model.PGPKeys

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbKeys)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restKeys := dbPGPKeysToREST(dbKeys)

		resp := map[string][]*api.GetPGPKeyRespObject{"pgpKeys": restKeys}
		handleError(w, logger, writeJSON(w, resp))
	}
}

func getPGPKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbKey, dbErr := retrieveDBPGPKey(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		restKey := dbPGPKeyToREST(dbKey)
		handleError(w, logger, writeJSON(w, restKey))
	}
}

func updatePGPKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbKey, dbErr := retrieveDBPGPKey(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		var restKey api.PatchPGPKeyReqObject
		if err := readJSON(r, &restKey); handleError(w, logger, err) {
			return
		}

		setIfValid(&dbKey.Name, restKey.Name)
		setIfValid(&dbKey.PublicKey, restKey.PublicKey)
		setIfValidSecret(&dbKey.PrivateKey, restKey.PrivateKey)

		if err := db.Update(dbKey).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbKey.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deletePGPKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbKey, dbErr := retrieveDBPGPKey(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		if err := db.Delete(dbKey).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
