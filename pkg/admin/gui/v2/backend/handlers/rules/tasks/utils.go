package tasks

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func listTasks(db database.ReadAccess, rule *model.Rule) (
	preTasks, postTasks, errorTasks model.Tasks, dbErr error,
) {
	if preTasks, dbErr = listTaskChain(db, rule.ID, model.ChainPre); dbErr != nil {
		return nil, nil, nil, dbErr
	}

	if postTasks, dbErr = listTaskChain(db, rule.ID, model.ChainPost); dbErr != nil {
		return nil, nil, nil, dbErr
	}

	if errorTasks, dbErr = listTaskChain(db, rule.ID, model.ChainError); dbErr != nil {
		return nil, nil, nil, dbErr
	}

	return preTasks, postTasks, errorTasks, nil
}

func listTaskChain(db database.ReadAccess, ruleID int64, chain model.Chain) (model.Tasks, error) {
	var tasks model.Tasks

	if err := db.Select(&tasks).
		Where("rule_id=?", ruleID).
		Where("chain=?", chain).
		OrderBy("rank", true).
		Run(); err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError, "failed to list tasks", err)
	}

	return tasks, nil
}

func getChain(r *http.Request) (model.Chain, error) {
	query := r.URL.Query()

	chain := query.Get("chain")
	if chain == "" {
		return "", common.NewError(http.StatusBadRequest, "missing task chain URL parameter")
	}

	switch model.Chain(chain) {
	case model.ChainPre, model.ChainPost, model.ChainError:
		return model.Chain(chain), nil
	default:
		return "", common.NewError(http.StatusBadRequest,
			fmt.Sprintf("invalid task chain URL parameter %q", chain))
	}
}

func getRank(r *http.Request) (int8, error) {
	query := r.URL.Query()

	rankStr := query.Get("rank")
	if rankStr == "" {
		return 0, common.NewError(http.StatusBadRequest, "missing task rank URL parameter")
	}

	rank, err := utils.ParseInt[int8](rankStr)
	if err != nil {
		return 0, common.NewErrorWith(http.StatusBadRequest, "invalid task rank URL parameter", err)
	}

	return rank, nil
}

func makeTask(db database.ReadAccess, r *http.Request) (*model.Task, error) {
	rule, rErr := rules.GetRule(db, r)
	if rErr != nil {
		return nil, rErr //nolint:wrapcheck //wrapping adds nothing here
	}

	chain, cErr := getChain(r)
	if cErr != nil {
		return nil, cErr
	}

	rank, rErr := getRank(r)
	if rErr != nil {
		return nil, rErr
	}

	return &model.Task{
		RuleID: rule.ID,
		Chain:  chain,
		Rank:   rank,
		Args:   make(map[string]string),
	}, nil
}

func getTask(db database.ReadAccess, r *http.Request) (*model.Task, error) {
	rule, rErr := rules.GetRule(db, r)
	if rErr != nil {
		return nil, rErr //nolint:wrapcheck //wrapping adds nothing here
	}

	chain, cErr := getChain(r)
	if cErr != nil {
		return nil, cErr
	}

	rank, rErr := getRank(r)
	if rErr != nil {
		return nil, rErr
	}

	var task model.Task
	if err := db.Get(&task, "rule_id=?", rule.ID).
		And("chain=?", chain).
		And("rank=?", rank).Run(); database.IsNotFound(err) {
		return nil, common.NewError(http.StatusNotFound, "task not found")
	} else if err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError, "failed to get task", err)
	}

	return &task, nil
}

func makeTaskForm(r *http.Request, forms map[string]*taskForm) (*taskForm, error) {
	query := r.URL.Query()

	taskType := query.Get("type")
	if taskType == "" {
		return nil, common.NewError(http.StatusBadRequest, "missing task type parameter")
	}

	formData := forms[taskType]
	if formData == nil {
		return nil, common.NewError(http.StatusInternalServerError,
			fmt.Sprintf("unknown task type %q", taskType))
	}

	return formData, nil
}

func parseTaskForm(db database.Access, r *http.Request) (*model.Task, error) {
	if err := r.ParseForm(); err != nil {
		return nil, common.NewErrorWith(http.StatusBadRequest, "failed to parse form", err)
	}

	if len(r.PostForm) == 0 {
		const maxMemory = 1024 * 1024
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			return nil, common.NewErrorWith(http.StatusBadRequest, "failed to parse multipart form", err)
		}
	}

	task, tErr := makeTask(db, r)
	if tErr != nil {
		return nil, tErr
	}

	const typeKey = "type"

	taskType := r.PostForm.Get(typeKey)
	if taskType == "" {
		return nil, common.NewError(http.StatusBadRequest, "missing task type input")
	}

	task.Type = taskType

	for key := range r.PostForm {
		if key != typeKey {
			if value := r.PostForm.Get(key); value != "" {
				task.Args[key] = value
			}
		}
	}

	return task, nil
}

func insertTask(db database.Access, task *model.Task) error {
	return db.Transaction(func(db *database.Session) error {
		if err := db.DeleteAll(task).
			Where("rule_id=?", task.RuleID).
			Where("chain=?", task.Chain).
			Where("rank=?", task.Rank).
			Run(); err != nil {
			return common.NewErrorWith(http.StatusInternalServerError,
				"failed to delete task", err)
		}

		if err := db.Insert(task).Run(); err != nil {
			return common.NewErrorWith(http.StatusInternalServerError, "failed to insert task", err)
		}

		return nil
	})
}

func deleteTask(db database.Access, task *model.Task) error {
	return db.Transaction(func(db *database.Session) error {
		if err := db.DeleteAll(task).
			Where("rule_id=?", task.RuleID).
			Where("chain=?", task.Chain).
			Where("rank=?", task.Rank).
			Run(); err != nil {
			return common.NewErrorWith(http.StatusInternalServerError,
				"failed to delete task", err)
		}

		if err := db.Exec(`UPDATE tasks SET rank=rank-1 WHERE rule_id=? AND
            chain=? AND rank>?`, task.RuleID, task.Chain, task.Rank); err != nil {
			return common.NewErrorWith(http.StatusInternalServerError,
				"failed to update task chain ranks", err)
		}

		return nil
	})
}

type reorderBody struct {
	RuleID int64       `json:"ruleID"`
	Chain  model.Chain `json:"chain"`
	Ranks  []int8      `json:"ranks"`
}

func reorderTasks(db database.Access, tasks model.Tasks, body reorderBody) error {
	return db.Transaction(func(db *database.Session) error {
		if err := db.DeleteAll(&model.Task{}).
			Where("rule_id=?", body.RuleID).
			Where("chain=?", body.Chain).
			Run(); err != nil {
			return common.NewErrorWith(http.StatusInternalServerError, "failed to delete chain", err)
		}

		for newRank, oldRank := range body.Ranks {
			task := tasks[oldRank]
			task.Rank = int8(newRank)

			if err := db.Insert(task).Run(); err != nil {
				return common.NewErrorWith(http.StatusInternalServerError, "failed to insert task", err)
			}
		}

		return nil
	})
}
