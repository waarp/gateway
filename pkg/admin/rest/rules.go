package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InRule is the JSON representation of a transfer rule in requests made to
// the REST interface.
type InRule struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
	IsSend  bool   `json:"isSend"`
	Path    string `json:"path"`
}

// ToModel transforms the JSON transfer rule into its database equivalent.
func (i *InRule) ToModel() *model.Rule {
	return &model.Rule{
		Name:    i.Name,
		Comment: i.Comment,
		IsSend:  i.IsSend,
		Path:    i.Path,
	}
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Comment string `json:"comment"`
	IsSend  bool   `json:"isSend"`
	Path    string `json:"path"`
}

// FromRule transforms the given database transfer rule into its JSON equivalent.
func FromRule(r *model.Rule) *OutRule {
	return &OutRule{
		ID:      r.ID,
		Name:    r.Name,
		Comment: r.Comment,
		IsSend:  r.IsSend,
		Path:    r.Path,
	}
}

// FromRules transforms the given list of database transfer rules into its JSON
// equivalent.
func FromRules(rs []model.Rule) []OutRule {
	rules := make([]OutRule, len(rs))
	for i, rule := range rs {
		rules[i] = OutRule{
			ID:      rule.ID,
			Name:    rule.Name,
			Comment: rule.Comment,
			IsSend:  rule.IsSend,
			Path:    rule.Path,
		}
	}
	return rules
}

func createRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonRule := &InRule{}
			if err := readJSON(r, jsonRule); err != nil {
				return err
			}

			rule := jsonRule.ToModel()
			if err := db.Create(rule); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, rule.ID))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "rule")
			if err != nil {
				return err
			}
			result := &model.Rule{ID: id}

			if err := get(db, result); err != nil {
				return err
			}

			return writeJSON(w, FromRule(result))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listRules(logger *log.Logger, db *database.Db) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseListFilters(r, validSorting)
			if err != nil {
				return err
			}

			var results []model.Rule
			if err := db.Select(&results, filters); err != nil {
				return err
			}

			resp := map[string][]OutRule{"rules": FromRules(results)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func updateRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "rule")
			if err != nil {
				return &notFound{}
			}

			if err := exist(db, &model.Rule{ID: id}); err != nil {
				return err
			}

			rule := &InRule{}
			if err := readJSON(r, rule); err != nil {
				return err
			}

			if err := db.Update(rule.ToModel(), id, false); err != nil {
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

func deleteRule(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			id, err := parseID(r, "rule")
			if err != nil {
				return &notFound{}
			}

			rule := &model.Rule{ID: id}
			if err := get(db, rule); err != nil {
				return err
			}

			if err := db.Delete(rule); err != nil {
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
