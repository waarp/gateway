package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

// InUser is the JSON representation of a user account in requests made to the
// REST interface.
type InUser struct {
	Username string `json:"username"`
	Password []byte `json:"password"`
}

func newInUser(old *model.User) *InUser {
	return &InUser{
		Username: old.Username,
		Password: old.Password,
	}
}

// ToModel transforms the JSON user into its database equivalent.
func (i *InUser) ToModel(id uint64) *model.User {
	return &model.User{
		ID:       id,
		Owner:    database.Owner,
		Username: i.Username,
		Password: i.Password,
	}
}

// OutUser is the JSON representation of a user account in responses sent by
// the REST interface.
type OutUser struct {
	Username string `json:"username"`
}

// FromUser transforms the given database user into its JSON equivalent.
func FromUser(user *model.User) *OutUser {
	return &OutUser{
		Username: user.Username,
	}
}

// FromUsers transforms the given list of user into its JSON equivalent.
func FromUsers(usr []model.User) []OutUser {
	users := make([]OutUser, len(usr))
	for i, user := range usr {
		users[i] = OutUser{
			Username: user.Username,
		}
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

			resp := map[string][]OutUser{"users": FromUsers(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonUser := &InUser{}
			if err := readJSON(r, jsonUser); err != nil {
				return err
			}

			user := jsonUser.ToModel(0)
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

			user := newInUser(old)
			if err := readJSON(r, user); err != nil {
				return err
			}

			if err := db.Update(user.ToModel(old.ID)); err != nil {
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

			user := &InUser{}
			if err := readJSON(r, user); err != nil {
				return err
			}

			if err := db.Update(user.ToModel(old.ID)); err != nil {
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
