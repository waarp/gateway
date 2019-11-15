package admin

import (
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/gorilla/mux"
)

func listTasks(logger *log.Logger, db *database.Db) http.HandlerFunc {
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

			preTasks := []model.Task{}
			preFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": ruleID, "chain": model.ChainPre},
			}
			if err := db.Select(&preTasks, preFilters); err != nil {
				return err
			}

			postTasks := []model.Task{}
			postFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": ruleID, "chain": model.ChainPost},
			}
			if err := db.Select(&postTasks, postFilters); err != nil {
				return err
			}

			errorTasks := []model.Task{}
			errorFilters := &database.Filters{
				Order:      "rank ASC",
				Conditions: builder.Eq{"rule_id": ruleID, "chain": model.ChainError},
			}
			if err := db.Select(&errorTasks, errorFilters); err != nil {
				return err
			}

			res := map[string][]model.Task{
				"preTasks":   preTasks,
				"postTasks":  postTasks,
				"errorTasks": errorTasks,
			}

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

func doTaskUpdate(ses *database.Session, req map[string][]*model.Task, ruleID uint64) error {
	if err := ses.Execute("DELETE FROM tasks WHERE rule_id=?", ruleID); err != nil {
		return err
	}
	for rank, task := range req["preTasks"] {
		task.RuleID = ruleID
		task.Chain = model.ChainPre
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, task := range req["postTasks"] {
		task.RuleID = ruleID
		task.Chain = model.ChainPost
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	for rank, task := range req["errorTasks"] {
		task.RuleID = ruleID
		task.Chain = model.ChainError
		task.Rank = uint32(rank)
		if err := ses.Create(task); err != nil {
			return err
		}
	}
	if err := ses.Commit(); err != nil {
		return err
	}

	return nil
}

func updateTasks(logger *log.Logger, db *database.Db) http.HandlerFunc {
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

			req := map[string][]*model.Task{}
			if err := readJSON(r, &req); err != nil {
				return err
			}

			ses, err := db.BeginTransaction()
			if err != nil {
				return err
			}
			if err := doTaskUpdate(ses, req, ruleID); err != nil {
				ses.Rollback()
				return err
			}

			w.Header().Set("Location", APIPath+RulesPath+"/"+mux.Vars(r)["rule"]+
				RuleTasksPath)
			w.WriteHeader(http.StatusCreated)

			return nil
		}()
		if res != nil {
			handleErrors(w, logger, res)
			return
		}
	}
}
