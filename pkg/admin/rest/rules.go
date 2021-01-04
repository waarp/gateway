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
	var rule model.Rule
	if err := db.Get(&rule, "name=? AND send=?", ruleName,
		ruleDirection == "send").Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("%s rule '%s' not found", ruleDirection, ruleName)
		}
		return nil, err
	}
	return &rule, nil
}

func addRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jsonRule api.InRule
		if err := readJSON(r, &jsonRule); handleError(w, logger, err) {
			return
		}

		rule, err := ruleToDB(&jsonRule, 0)
		if handleError(w, logger, err) {
			return
		}

		err = db.Transaction(func(ses *database.Session) database.Error {
			if err := ses.Insert(rule).Run(); err != nil {
				return err
			}

			if err := doTaskUpdate(ses, jsonRule.UptRule, rule.ID, true); err != nil {
				return err
			}

			return nil
		})
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, rule.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getRl(r, db)
		if handleError(w, logger, err) {
			return
		}

		rule, err := FromRule(db, result)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, rule)
		handleError(w, logger, err)
	}
}

func listRules(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var rules model.Rules
		query, err := parseSelectQuery(r, db, validSorting, &rules)
		if handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		jRules, err := FromRules(db, rules)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutRule{"rules": jRules}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func updateRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getRl(r, db)
		if handleError(w, logger, err) {
			return
		}

		jRule := newInRule(old)
		if err := readJSON(r, jRule); handleError(w, logger, err) {
			return
		}

		rule, err := ruleToDB(jRule, old.ID)
		if handleError(w, logger, err) {
			return
		}

		err = db.Transaction(func(ses *database.Session) database.Error {
			if err := ses.Update(rule).Run(); err != nil {
				return err
			}

			if err := doTaskUpdate(ses, jRule.UptRule, old.ID, false); err != nil {
				return err
			}

			return nil
		})
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(jRule.Name)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getRl(r, db)
		if handleError(w, logger, err) {
			return
		}

		jRule := &api.InRule{IsSend: &old.IsSend, UptRule: &api.UptRule{}}
		if err := readJSON(r, jRule.UptRule); handleError(w, logger, err) {
			return
		}

		rule, err := ruleToDB(jRule, old.ID)
		if handleError(w, logger, err) {
			return
		}

		err = db.Transaction(func(ses *database.Session) database.Error {
			if err := ses.Update(rule).Run(); handleError(w, logger, err) {
				return err
			}

			if err := doTaskUpdate(ses, jRule.UptRule, old.ID, true); err != nil {
				return err
			}

			return nil
		})
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(jRule.Name)))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rule, err := getRl(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(rule).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func allowAllRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rule, err := getRl(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = db.DeleteAll(&model.RuleAccess{}).Where("rule_id=?", rule.ID).Run()
		if handleError(w, logger, err) {
			return
		}

		fmt.Fprintf(w, "Usage of the %s rule '%s' is now unrestricted.",
			ruleDirection(rule), rule.Name)
	}
}
