package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

// ruleToDB transforms the JSON transfer rule into its database equivalent.
func ruleToDB(rule *api.InRule, id uint64) (*model.Rule, error) {
	if rule.IsSend == nil {
		return nil, badRequest("missing rule direction")
	}
	return &model.Rule{
		ID:       id,
		Name:     str(rule.Name),
		Comment:  str(rule.Comment),
		IsSend:   *rule.IsSend,
		Path:     str(rule.Path),
		InPath:   str(rule.InPath),
		OutPath:  str(rule.OutPath),
		WorkPath: str(rule.WorkPath),
	}, nil
}

func newInRule(old *model.Rule) *api.InRule {
	return &api.InRule{
		UptRule: &api.UptRule{
			Name:     &old.Name,
			Comment:  &old.Comment,
			Path:     &old.Path,
			InPath:   &old.InPath,
			OutPath:  &old.OutPath,
			WorkPath: &old.WorkPath,
		},
		IsSend: &old.IsSend,
	}
}

// FromRule transforms the given database transfer rule into its JSON equivalent.
func FromRule(db *database.DB, r *model.Rule) (*api.OutRule, error) {
	access, err := makeRuleAccess(db, r)
	if err != nil {
		return nil, err
	}

	rule := &api.OutRule{
		Name:       r.Name,
		Comment:    r.Comment,
		IsSend:     r.IsSend,
		Path:       r.Path,
		InPath:     r.InPath,
		OutPath:    r.OutPath,
		WorkPath:   r.WorkPath,
		Authorized: access,
	}
	if err := doListTasks(db, rule, r.ID); err != nil {
		return nil, err
	}
	return rule, nil
}

// FromRules transforms the given list of database transfer rules into its JSON
// equivalent.
func FromRules(db *database.DB, rs []model.Rule) ([]api.OutRule, error) {
	rules := make([]api.OutRule, len(rs))
	for i, r := range rs {
		rule := r
		res, err := FromRule(db, &rule)
		if err != nil {
			return nil, err
		}
		rules[i] = *res
	}
	return rules, nil
}

func ruleDirection(rule *model.Rule) string {
	if rule.IsSend {
		return "send"
	}
	return "receive"
}

func getRl(r *http.Request, db *database.DB) (*model.Rule, error) {
	ruleName, ok := mux.Vars(r)["rule"]
	if !ok {
		return nil, notFound("missing rule name")
	}
	ruleDirection, ok := mux.Vars(r)["direction"]
	if !ok {
		return nil, notFound("missing rule direction")
	}
	rule := &model.Rule{Name: ruleName, IsSend: ruleDirection == "send"}
	if err := db.Get(rule); err != nil {
		if err == database.ErrNotFound {
			return nil, notFound("rule '%s' not found", ruleName)
		}
		return nil, err
	}
	return rule, nil
}

func createRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonRule := &api.InRule{}
			if err := readJSON(r, jsonRule); err != nil {
				return err
			}

			rule, err := ruleToDB(jsonRule, 0)
			if err != nil {
				return err
			}
			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			if err := ses.Create(rule); err != nil {
				ses.Rollback()
				return err
			}
			if err := doTaskUpdate(ses, jsonRule.UptRule, rule.ID, true); err != nil {
				ses.Rollback()
				return err
			}
			if err := ses.Commit(); err != nil {
				return err
			}

			w.Header().Set("Location", location(r.URL, rule.Name))
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

			rule, err := FromRule(db, result)
			if err != nil {
				return err
			}

			return writeJSON(w, rule)
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

			rules, err := FromRules(db, results)
			if err != nil {
				return err
			}

			resp := map[string][]api.OutRule{"rules": rules}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func updateRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getRl(r, db)
			if err != nil {
				return err
			}

			jRule := newInRule(old)
			if err := readJSON(r, jRule); err != nil {
				return err
			}

			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			rule, err := ruleToDB(jRule, old.ID)
			if err != nil {
				return err
			}
			if err := ses.Update(rule); err != nil {
				ses.Rollback()
				return err
			}
			if err := doTaskUpdate(ses, jRule.UptRule, old.ID, false); err != nil {
				ses.Rollback()
				return err
			}
			if err := ses.Commit(); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, str(jRule.Name)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func replaceRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			old, err := getRl(r, db)
			if err != nil {
				return err
			}

			jRule := &api.InRule{IsSend: &old.IsSend, UptRule: &api.UptRule{}}
			if err := readJSON(r, jRule.UptRule); err != nil {
				return err
			}

			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			rule, err := ruleToDB(jRule, old.ID)
			if err != nil {
				return err
			}
			if err := ses.Update(rule); err != nil {
				ses.Rollback()
				return err
			}
			if err := doTaskUpdate(ses, jRule.UptRule, old.ID, true); err != nil {
				ses.Rollback()
				return err
			}
			if err := ses.Commit(); err != nil {
				return err
			}

			w.Header().Set("Location", locationUpdate(r.URL, str(jRule.Name)))
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

func allowAllRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			rule, err := getRl(r, db)
			if err != nil {
				return err
			}

			if err := db.Execute("DELETE FROM rule_access WHERE rule_id=?", rule.ID); err != nil {
				return err
			}

			fmt.Fprintf(w, "Usage of the %s rule '%s' is now unrestricted.",
				ruleDirection(rule), rule.Name)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
