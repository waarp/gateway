package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func newInUser(old *model.User) *api.InUser {
	return &api.InUser{
		Username: &old.Username,
		Password: strPtr(string(old.Password)),
	}
}

// userToDB transforms the JSON user into its database equivalent.
func userToDB(user *api.InUser, old *model.User) (*model.User, error) {
	mask, err := permsToMask(old.Permissions, user.Perms)
	if err != nil {
		return nil, err
	}

	return &model.User{
		ID:          old.ID,
		Owner:       conf.GlobalConfig.GatewayName,
		Username:    str(user.Username),
		Password:    []byte(str(user.Password)),
		Permissions: mask,
	}, nil
}

// FromUser transforms the given database user into its JSON equivalent.
func FromUser(user *model.User) *api.OutUser {
	return &api.OutUser{
		Username: user.Username,
		Perms:    *maskToPerms(user.Permissions),
	}
}

func writeUsers(users model.Users, w http.ResponseWriter) error {
	jUsers := make([]api.OutUser, len(users))
	for i := range users {
		jUsers[i] = *FromUser(&users[i])
	}
	return writeJSON(w, map[string][]api.OutUser{"users": jUsers})
}

func getUsr(r *http.Request, db *database.DB) (*model.User, error) {
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
		return nil, err
	}
	return &user, nil
}

func getUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getUsr(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, FromUser(result)))
	}
}

func listUsers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":   order{"username", true},
		"username+": order{"username", true},
		"username-": order{"username", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var users model.Users
		query, err := parseSelectQuery(r, db, validSorting, &users)
		if handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeUsers(users, w))
	}
}

func addUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jsonUser api.InUser
		if err := readJSON(r, &jsonUser); handleError(w, logger, err) {
			return
		}

		user, err := userToDB(&jsonUser, &model.User{})
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(user).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, user.Username))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getUsr(r, db)
		if handleError(w, logger, err) {
			return
		}

		jUser := newInUser(old)
		if err := readJSON(r, jUser); handleError(w, logger, err) {
			return
		}

		user, err := userToDB(jUser, old)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Update(user).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(jUser.Username)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getUsr(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jUser api.InUser
		if err := readJSON(r, &jUser); handleError(w, logger, err) {
			return
		}

		user, err := userToDB(&jUser, old)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Update(user).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(jUser.Username)))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getUsr(r, db)
		if handleError(w, logger, err) {
			return
		}

		login, _, _ := r.BasicAuth()
		if user.Username == login {
			handleError(w, logger, &forbidden{"user cannot delete self"})
			return
		}

		if err := db.Delete(user).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
