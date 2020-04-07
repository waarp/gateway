package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

// InUser is the JSON representation of a user account in requests made to the
// REST interface.
type InUser struct {
	Username string `json:"username"`
	Password []byte `json:"password"`
}

// ToModel transforms the JSON user into its database equivalent.
func (i *InUser) ToModel() *model.User {
	return &model.User{
		Username: i.Username,
		Password: i.Password,
	}
}

// OutUser is the JSON representation of a user account in responses sent by
// the REST interface.
type OutUser struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// FromUser transforms the given database user into its JSON equivalent.
func FromUser(user *model.User) *OutUser {
	return &OutUser{
		ID:       user.ID,
		Username: user.Username,
	}
}

// FromUsers transforms the given list of user into its JSON equivalent.
func FromUsers(usr []model.User) []OutUser {
	users := make([]OutUser, len(usr))
	for i, user := range usr {
		users[i] = OutUser{
			ID:       user.ID,
			Username: user.Username,
		}
	}
	return users
}

func getUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "user")
			if err != nil {
				return &notFound{}
			}
			result := &model.User{ID: id, Owner: database.Owner}

			if err := get(db, result); err != nil {
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

			user := jsonUser.ToModel()
			if err := db.Create(user); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, user.ID))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func updateUser(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "user")
			if err != nil {
				return &notFound{}
			}

			if err := exist(db, &model.User{ID: id, Owner: database.Owner}); err != nil {
				return err
			}

			user := &InUser{}
			if err := readJSON(r, user); err != nil {
				return err
			}

			if err := db.Update(user.ToModel(), id, false); err != nil {
				return err
			}

			w.Header().Set("Location", location(r))
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
			id, err := parseID(r, "user")
			if err != nil {
				return &notFound{}
			}

			user := &model.User{ID: id, Owner: database.Owner}
			if err := get(db, user); err != nil {
				return err
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
