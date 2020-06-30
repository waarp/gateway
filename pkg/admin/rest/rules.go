package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

// InRule is the JSON representation of a transfer rule in requests made to
// the REST interface.
type InRule struct {
	*UptRule
	IsSend bool `json:"isSend"`
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

// UptRule is the JSON representation of a transfer rule in updated requests made to
// the REST interface.
type UptRule struct {
	Name       string     `json:"name"`
	Comment    string     `json:"comment"`
	Path       string     `json:"path"`
	InPath     string     `json:"inPath"`
	OutPath    string     `json:"outPath"`
	WorkPath   string     `json:"workPath"`
	PreTasks   []RuleTask `json:"preTasks"`
	PostTasks  []RuleTask `json:"postTasks"`
	ErrorTasks []RuleTask `json:"errorTasks"`
}

// ToModel transforms the JSON transfer rule into its database equivalent.
func (i *UptRule) ToModel() *model.Rule {
	return &model.Rule{
		Name:     i.Name,
		Comment:  i.Comment,
		Path:     i.Path,
		InPath:   i.InPath,
		OutPath:  i.OutPath,
		WorkPath: i.WorkPath,
	}
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	Name       string      `json:"name"`
	Comment    string      `json:"comment"`
	IsSend     bool        `json:"isSend"`
	Path       string      `json:"path"`
	InPath     string      `json:"inPath"`
	OutPath    string      `json:"outPath"`
	WorkPath   string      `json:"workPath"`
	Authorized *RuleAccess `json:"authorized"`
	PreTasks   []RuleTask  `json:"preTasks"`
	PostTasks  []RuleTask  `json:"postTasks"`
	ErrorTasks []RuleTask  `json:"errorTasks"`
}

// FromRule transforms the given database transfer rule into its JSON equivalent.
func FromRule(db *database.DB, r *model.Rule) (*OutRule, error) {
	access, err := makeRuleAccess(db, r)
	if err != nil {
		return nil, err
	}

	rule := &OutRule{
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
func FromRules(db *database.DB, rs []model.Rule) ([]OutRule, error) {
	rules := make([]OutRule, len(rs))
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
			jsonRule := &InRule{}
			if err := readJSON(r, jsonRule); err != nil {
				return err
			}

			rule := jsonRule.ToModel()
			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			if err := ses.Create(rule); err != nil {
				ses.Rollback()
				return err
			}
			if err := doTaskUpdate(ses, jsonRule.UptRule, rule.ID); err != nil {
				ses.Rollback()
				return err
			}
			if err := ses.Commit(); err != nil {
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

			resp := map[string][]OutRule{"rules": rules}
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
			check, err := getRl(r, db)
			if err != nil {
				return err
			}

			rule := &UptRule{}
			if err := readJSON(r, rule); err != nil {
				return err
			}

			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			if err := ses.Update(rule.ToModel(), check.ID, false); err != nil {
				ses.Rollback()
				return err
			}
			if err := doTaskUpdate(ses, rule, check.ID); err != nil {
				ses.Rollback()
				return err
			}
			if err := ses.Commit(); err != nil {
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

			http.Error(w, fmt.Sprintf("Access to rule '%s' is now unrestricted.",
				rule.Name), http.StatusOK)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}
