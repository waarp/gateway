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

func retrieveDBCryptoKey(r *http.Request, db database.ReadAccess) (*model.CryptoKey, error) {
	keyName, ok := mux.Vars(r)["crypto_key"]
	if !ok {
		return nil, notFound("missing cryptographic key name")
	}

	var dbKey model.CryptoKey
	if err := db.Get(&dbKey, "name=?", keyName).Owner().Run(); database.IsNotFound(err) {
		return nil, notFound("Cryptographic key %q not found", keyName)
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	return &dbKey, nil
}

func dbCryptoKeyToREST(dbKey *model.CryptoKey) *api.GetCryptoKeyRespObject {
	return &api.GetCryptoKeyRespObject{
		Name: dbKey.Name,
		Type: dbKey.Type,
		Key:  dbKey.Key.String(),
	}
}

func dbCryptoKeysToREST(dbKeys []*model.CryptoKey) []*api.GetCryptoKeyRespObject {
	restKeys := make([]*api.GetCryptoKeyRespObject, len(dbKeys))
	for i := range dbKeys {
		restKeys[i] = dbCryptoKeyToREST(dbKeys[i])
	}

	return restKeys
}

func addCryptoKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restKey api.PostCryptoKeyReqObject
		if err := readJSON(r, &restKey); handleError(w, logger, err) {
			return
		}

		dbKey := &model.CryptoKey{
			Name: restKey.Name,
			Type: restKey.Type,
			Key:  database.SecretText(restKey.Key),
		}
		if err := db.Insert(dbKey).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbKey.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for a different type, keep separate
func listCryptoKeys(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
		"type+":   order{"type", true},
		"type-":   order{"type", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbKeys model.CryptoKeys

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbKeys)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Owner().Run(); handleError(w, logger, err) {
			return
		}

		restKeys := dbCryptoKeysToREST(dbKeys)

		resp := map[string][]*api.GetCryptoKeyRespObject{"cryptoKeys": restKeys}
		handleError(w, logger, writeJSON(w, resp))
	}
}

func getCryptoKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbKey, dbErr := retrieveDBCryptoKey(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		restKey := dbCryptoKeyToREST(dbKey)
		handleError(w, logger, writeJSON(w, restKey))
	}
}

func updateCryptoKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbKey, dbErr := retrieveDBCryptoKey(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		var restKey api.PatchCryptoKeyReqObject
		if err := readJSON(r, &restKey); handleError(w, logger, err) {
			return
		}

		setIfValid(&dbKey.Name, restKey.Name)
		setIfValid(&dbKey.Type, restKey.Type)
		setIfValidSecret(&dbKey.Key, restKey.Key)

		if err := db.Update(dbKey).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbKey.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteCryptoKey(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbKey, dbErr := retrieveDBCryptoKey(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		if err := db.Delete(dbKey).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
