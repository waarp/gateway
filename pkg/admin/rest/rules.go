package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

// InRule is the JSON representation of a transfer rule in requests made to
// the REST interface.
type InRule struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
	IsSend  bool   `json:"isSend"`
	Path    string `json:"path"`
	InPath  string `json:"inPath"`
	OutPath string `json:"outPath"`
}

// ToModel transforms the JSON transfer rule into its database equivalent.
func (i *InRule) ToModel() *model.Rule {
	return &model.Rule{
		Name:    i.Name,
		Comment: i.Comment,
		IsSend:  i.IsSend,
		Path:    i.Path,
		InPath:  i.InPath,
		OutPath: i.OutPath,
	}
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	Name       string     `json:"name"`
	Comment    string     `json:"comment"`
	IsSend     bool       `json:"isSend"`
	Path       string     `json:"path"`
	InPath     string     `json:"inPath"`
	OutPath    string     `json:"outPath"`
	Authorized RuleAccess `json:"authorized"`
}

// FromRule transforms the given database transfer rule into its JSON equivalent.
func FromRule(r *model.Rule, access *RuleAccess) *OutRule {
	return &OutRule{
		Name:       r.Name,
		Comment:    r.Comment,
		IsSend:     r.IsSend,
		Path:       r.Path,
		InPath:     r.InPath,
		OutPath:    r.OutPath,
		Authorized: *access,
	}
}

// FromRules transforms the given list of database transfer rules into its JSON
// equivalent.
func FromRules(rs []model.Rule, accesses map[uint64]RuleAccess) []OutRule {
	rules := make([]OutRule, len(rs))
	for i, r := range rs {
		rule := r
		access := accesses[rule.ID]
		rules[i] = *FromRule(&rule, &access)
	}
	return rules
}

func getRl(r *http.Request, db *database.DB) (*model.Rule, error) {
	ruleName, ok := mux.Vars(r)["rule"]
	if !ok {
		return nil, &notFound{}
	}
	rule := &model.Rule{Name: ruleName}
	if err := get(db, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func createRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
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

			w.Header().Set("Location", location(r, rule.Name))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getRl(r, db)
			if err != nil {
				return err
			}

			access, err := makeRuleAccess(db, result)
			if err != nil {
				return err
			}

			return writeJSON(w, FromRule(result, access))
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listRules(logger *log.Logger, db *database.DB) http.HandlerFunc {
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

			accesses, err := makeRulesAccesses(db, results)
			if err != nil {
				return err
			}

			resp := map[string][]OutRule{"rules": FromRules(results, accesses)}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

//nolint:dupl
func updateRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getRl(r, db)
			if err != nil {
				return err
			}

			rule := &InRule{}
			if err := readJSON(r, rule); err != nil {
				return err
			}

			if err := db.Update(rule.ToModel(), check.ID, false); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r, rule.Name, check.Name))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func deleteRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			rule, err := getRl(r, db)
			if err != nil {
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

func restrictRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			rule, err := getRl(r, db)
			if err != nil {
				return err
			}

			if err := db.Execute(builder.Delete(builder.Eq{"rule_id": rule.ID}).
				From("rule_access")); err != nil {
				return err
			}

			http.Error(w, fmt.Sprintf("Access to rule '%s' is now restricted.",
				rule.Name), http.StatusOK)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
