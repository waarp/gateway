package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

// InRuleAccess is the JSON representation of a rule access in requests made to
// the REST interface.
type InRuleAccess struct {
	ObjectID   uint64 `json:"objectID"`
	ObjectType string `json:"objectType"`
}

// ToModel transforms the JSON rule access into its database equivalent.
func (i *InRuleAccess) ToModel() *model.RuleAccess {
	return &model.RuleAccess{
		ObjectID:   i.ObjectID,
		ObjectType: i.ObjectType,
	}
}

// OutRuleAccess is the JSON representation of a rule access in responses sent by
// the REST interface.
type OutRuleAccess struct {
	ObjectID   uint64 `json:"objectID"`
	ObjectType string `json:"objectType"`
}

// FromRuleAccess transforms the given list of database rule accesses into its
// JSON equivalent.
func FromRuleAccess(as []model.RuleAccess) []OutRuleAccess {
	accesses := make([]OutRuleAccess, len(as))
	for i, acc := range as {
		accesses[i] = OutRuleAccess{
			ObjectID:   acc.ObjectID,
			ObjectType: acc.ObjectType,
		}
	}
	return accesses
}

func createAccess(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			ruleID, err := parseID(r, "rule")
			if err != nil {
				return err
			}
			if ok, err := db.Exists(&model.Rule{ID: ruleID}); err != nil {
				return err
			} else if !ok {
				return &notFound{}
			}

			jsonAccess := &InRuleAccess{}
			if err := readJSON(r, jsonAccess); err != nil {
				return err
			}

			ok, err := db.Exists(&model.RuleAccess{RuleID: ruleID})
			if err != nil {
				return err
			}

			access := jsonAccess.ToModel()
			access.RuleID = ruleID
			if err := db.Create(access); err != nil {
				return err
			}

			w.Header().Set("Location", location(r))
			if !ok {
				http.Error(w, fmt.Sprintf("Access to rule %v is now restricted.",
					ruleID), http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusCreated)
			}
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listAccess(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := parseID(r, "rule")
			if err != nil {
				return err
			}

			if ok, err := db.Exists(&model.Rule{ID: ruleID}); err != nil {
				return err
			} else if !ok {
				return &notFound{}
			}

			acc := []model.RuleAccess{}
			filters := &database.Filters{Conditions: builder.Eq{"rule_id": ruleID}}
			if err := db.Select(&acc, filters); err != nil {
				return err
			}

			res := map[string][]OutRuleAccess{}
			res["permissions"] = FromRuleAccess(acc)
			if err := writeJSON(w, res); err != nil {
				return err
			}

			w.WriteHeader(http.StatusOK)
			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
		}
	}
}

func deleteAccess(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := parseID(r, "rule")
			if err != nil {
				return err
			}

			jsonAcc := &InRuleAccess{}
			if err := readJSON(r, jsonAcc); err != nil {
				return err
			}

			acc := jsonAcc.ToModel()
			acc.RuleID = ruleID
			if err := get(db, acc); err != nil {
				return err
			}

			if err := db.Delete(acc); err != nil {
				return err
			}

			ok, err := db.Exists(&model.RuleAccess{RuleID: ruleID})
			if err != nil {
				return err
			}
			if !ok {
				http.Error(w, fmt.Sprintf("Access to rule %v is now unrestricted.",
					ruleID), http.StatusOK)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
		}

	}
}
