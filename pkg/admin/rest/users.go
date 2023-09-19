package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func dbUserToRESTInput(old *model.User) *api.InUser {
	return &api.InUser{
		Username: &old.Username,
		Password: strPtr(old.PasswordHash),
	}
}

// restUserToDB transforms the JSON user into its database equivalent.
func restUserToDB(user *api.InUser, old *model.User) (*model.User, error) {
	mask, err := permsToMask(old.Permissions, user.Perms)
	if err != nil {
		return nil, err
	}

	hash, err := utils.HashPassword(database.BcryptRounds, str(user.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to hash passwordi: %w", err)
	}

	return &model.User{
		ID:           old.ID,
		Owner:        conf.GlobalConfig.GatewayName,
		Username:     str(user.Username),
		PasswordHash: hash,
		Permissions:  mask,
	}, nil
}

// DBUserToREST transforms the given database user into its JSON equivalent.
func DBUserToREST(dbUser *model.User) *api.OutUser {
	return &api.OutUser{
		Username: dbUser.Username,
		Perms:    *maskToPerms(dbUser.Permissions),
	}
}

// DBUsersToREST transforms the given database users into their JSON equivalents.
func DBUsersToREST(dbUsers []*model.User) []*api.OutUser {
	restUsers := make([]*api.OutUser, len(dbUsers))

	for i, dbUser := range dbUsers {
		restUsers[i] = DBUserToREST(dbUser)
	}

	return restUsers
}

//nolint:dupl //duplicate is for a completely different type (servers), keep separate
func retrieveDBUser(r *http.Request, db *database.DB) (*model.User, error) {
	username, ok := mux.Vars(r)["user"]
	if !ok {
		return nil, notFound("missing username")
	}

	var user model.User
	if err := db.Get(&user, "username=? AND owner=?", username, conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("user '%s' not found", username)
		}

		return nil, fmt.Errorf("failed to retrieve user %q: %w", username, err)
	}

	return &user, nil
}

func getUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbUser, getErr := retrieveDBUser(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restUser := DBUserToREST(dbUser)
		handleError(w, logger, writeJSON(w, restUser))
	}
}

func listUsers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":   order{"username", true},
		"username+": order{"username", true},
		"username-": order{"username", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbUsers model.Users

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbUsers)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Where("owner=?", conf.GlobalConfig.GatewayName).
			Run(); handleError(w, logger, err) {
			return
		}

		restUsers := DBUsersToREST(dbUsers)
		response := map[string][]*api.OutUser{"users": restUsers}

		handleError(w, logger, writeJSON(w, response))
	}
}

func addUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restUser api.InUser
		if err := readJSON(r, &restUser); handleError(w, logger, err) {
			return
		}

		dbUser, err := restUserToDB(&restUser, &model.User{})
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(dbUser).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbUser.Username))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldUser, getErr := retrieveDBUser(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restUser := dbUserToRESTInput(oldUser)
		if err := readJSON(r, restUser); handleError(w, logger, err) {
			return
		}

		dbUser, convErr := restUserToDB(restUser, oldUser)
		if handleError(w, logger, convErr) {
			return
		}

		dbUser.ID = oldUser.ID
		if err := db.Update(dbUser).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(restUser.Username)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldUser, getErr := retrieveDBUser(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restUser api.InUser
		if err := readJSON(r, &restUser); handleError(w, logger, err) {
			return
		}

		dbUser, convErr := restUserToDB(&restUser, oldUser)
		if handleError(w, logger, convErr) {
			return
		}

		dbUser.ID = oldUser.ID
		if err := db.Update(dbUser).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(restUser.Username)))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbUser, getErr := retrieveDBUser(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		login, _, _ := r.BasicAuth()
		if dbUser.Username == login {
			handleError(w, logger, &forbidden{"a user cannot delete themself"})

			return
		}

		if err := db.Delete(dbUser).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
