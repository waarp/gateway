package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
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
		Owner:       database.Owner,
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

// FromUsers transforms the given list of user into its JSON equivalent.
func FromUsers(usr []model.User) []api.OutUser {
	users := make([]api.OutUser, len(usr))
	for i := range usr {
		users[i] = *FromUser(&usr[i])
	}
	return users
}

func getUsr(r *http.Request, db *database.DB) (*model.User, error) {
	username, ok := mux.Vars(r)["user"]
	if !ok {
		return nil, notFound("missing username")
	}
	user := &model.User{Username: username, Owner: database.Owner}
	if err := db.Get(user); err != nil {
		return nil, notFound("user '%s' not found", username)
	}
	return user, nil
}

func getUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getUsr(r, db)
			if err != nil {
				return err
			}

			return writeJSON(w, FromUser(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listUsers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default":   "username ASC",
		"username+": "username ASC",
		"username-": "username DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}
			filters.Conditions = builder.Eq{"owner": database.Owner}

			var results []model.User
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]api.OutUser{"users": FromUsers(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func addUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonUser := &api.InUser{}
			if err := readJSON(r, jsonUser); err != nil {
				return err
			}

			user, err := userToDB(jsonUser, &model.User{})
			if err != nil {
				return err
			}
			if err := db.Create(user); err != nil {
				return err
			}

			w.Header().Set("Location", location(r.URL, user.Username))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func updateUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getUsr(r, db)
			if err != nil {
				return err
			}

			jUser := newInUser(old)
			if err := readJSON(r, jUser); err != nil {
				return err
			}

			user, err := userToDB(jUser, old)
			if err != nil {
				return err
			}
			if err := db.Update(user); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, user.Username))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func replaceUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getUsr(r, db)
			if err != nil {
				return err
			}

			jUser := &api.InUser{}
			if err := readJSON(r, jUser); err != nil {
				return err
			}

			user, err := userToDB(jUser, old)
			if err != nil {
				return err
			}

			if err := db.Update(user); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, user.Username))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			user, err := getUsr(r, db)
			if err != nil {
				return err
			}

			login, _, _ := r.BasicAuth()
			if user.Username == login {
				return &forbidden{msg: "user cannot delete self"}
			}

			if err := db.Delete(user); err != nil {
				return err
			}
			w.WriteHeader(http.StatusNoContent)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
