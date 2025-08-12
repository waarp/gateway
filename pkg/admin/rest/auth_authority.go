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

func authorityToModel(jAuthority *api.InAuthority, id int64) *model.Authority {
	return &model.Authority{
		ID:             id,
		Name:           jAuthority.Name.Value,
		Type:           jAuthority.Type.Value,
		PublicIdentity: jAuthority.PublicIdentity.Value,
		ValidHosts:     jAuthority.ValidHosts,
	}
}

func modelToAuthority(dbAuthority *model.Authority) *api.OutAuthority {
	return &api.OutAuthority{
		Name:           dbAuthority.Name,
		Type:           dbAuthority.Type,
		PublicIdentity: dbAuthority.PublicIdentity,
		ValidHosts:     dbAuthority.ValidHosts,
	}
}

func modelToAuthorities(dbAuthorities model.Authorities) []*api.OutAuthority {
	jAuthorities := make([]*api.OutAuthority, 0, len(dbAuthorities))

	for _, dbAuthority := range dbAuthorities {
		jAuthorities = append(jAuthorities, modelToAuthority(dbAuthority))
	}

	return jAuthorities
}

func getAuthority(r *http.Request, db database.ReadAccess) (*model.Authority, error) {
	authName, ok := mux.Vars(r)["authority"]
	if !ok {
		return nil, notFound("missing authority name")
	}

	var authority model.Authority
	if err := db.Get(&authority, "name=?", authName).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve authority %q: %w", authName, err)
	}

	return &authority, nil
}

//nolint:dupl //partners and authorities have nothing in common
func addAuthAuthority(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jAuth api.InAuthority
		if err := readJSON(r, &jAuth); handleError(w, logger, err) {
			return
		}

		dbAuth := authorityToModel(&jAuth, 0)
		if err := db.Insert(dbAuth).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbAuth.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getAuthAuthority(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbAuth, err := getAuthority(r, db)
		if handleError(w, logger, err) {
			return
		}

		jAuth := modelToAuthority(dbAuth)
		handleError(w, logger, writeJSON(w, jAuth))
	}
}

func listAuthAuthorities(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbAuthorities model.Authorities

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbAuthorities)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		jAuthorities := modelToAuthorities(dbAuthorities)
		resp := map[string][]*api.OutAuthority{"authorities": jAuthorities}

		handleError(w, logger, writeJSON(w, resp))
	}
}

//nolint:dupl //partners and authorities have nothing in common
func updateAuthAuthority(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, dbErr := getAuthority(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		jAuth := &api.InAuthority{
			Name:           asNullable(old.Name),
			Type:           asNullable(old.Type),
			PublicIdentity: asNullable(old.PublicIdentity),
			ValidHosts:     old.ValidHosts,
		}
		if err := readJSON(r, jAuth); handleError(w, logger, err) {
			return
		}

		dbAuth := authorityToModel(jAuth, old.ID)
		if err := db.Update(dbAuth).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAuth.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceAuthAuthority(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, dbErr := getAuthority(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		var jAuth api.InAuthority
		if err := readJSON(r, &jAuth); handleError(w, logger, err) {
			return
		}

		dbAuth := authorityToModel(&jAuth, old.ID)
		if err := db.Update(dbAuth).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbAuth.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteAuthAuthority(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, dbErr := getAuthority(r, db)
		if handleError(w, logger, dbErr) {
			return
		}

		if err := db.Delete(old).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
