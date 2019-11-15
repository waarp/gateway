package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func createAccess(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := strconv.ParseUint(mux.Vars(r)["rule"], 10, 64)
			if err != nil {
				return &notFound{}
			}

			if ok, err := db.Exists(&model.Rule{ID: ruleID}); err != nil {
				return err
			} else if !ok {
				return &notFound{}
			}

			acc := &model.RuleAccess{RuleID: ruleID}
			if err := readJSON(r, acc); err != nil {
				return err
			}
			if err := db.Create(acc); err != nil {
				return err
			}

			res, err := db.Query("SELECT * FROM rule_access WHERE rule_id=?", ruleID)
			if err != nil {
				return err
			}

			w.Header().Set("Location", APIPath+RulesPath+"/"+mux.Vars(r)["rule"]+
				RulePermissionPath)
			w.WriteHeader(http.StatusCreated)
			if len(res) == 1 {
				msg := fmt.Sprintf("Access to rule %v is now restricted", ruleID)
				fmt.Fprintln(w, msg)
			}

			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
			return
		}
	}
}

func listAccess(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := strconv.ParseUint(mux.Vars(r)["rule"], 10, 64)
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

			res := map[string][]model.RuleAccess{}
			res["permissions"] = acc
			if err := writeJSON(w, res); err != nil {
				return err
			}

			w.WriteHeader(http.StatusOK)
			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
			return
		}
	}
}

func deleteAccess(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := func() error {
			ruleID, err := strconv.ParseUint(mux.Vars(r)["rule"], 10, 64)
			if err != nil {
				return &notFound{}
			}

			acc := &model.RuleAccess{RuleID: ruleID}
			if err := readJSON(r, acc); err != nil {
				return err
			}
			if err := db.Delete(acc); err != nil {
				if err == database.ErrNotFound {
					return &notFound{}
				}
				return err
			}

			res, err := db.Query("SELECT * FROM rule_access WHERE rule_id=?", ruleID)
			if err != nil {
				return err
			}

			w.WriteHeader(http.StatusNoContent)
			if len(res) == 0 {
				msg := fmt.Sprintf("Access to rule %v is now unrestricted", ruleID)
				fmt.Fprintln(w, msg)
			}

			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
			return
		}

	}
}
