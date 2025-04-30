package internal

import (
	"maps"
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func ValidTasks() []string {
	return slices.Collect(maps.Keys(model.ValidTasks))
}

func listTasks(db database.ReadAccess, rule *model.Rule, chain model.Chain) ([]*model.Task, error) {
	var tasks model.Tasks

	return tasks, db.Select(&tasks).Where("rule_id=? AND chain=?", rule.ID, chain).
		OrderBy("rank", true).Run()
}

func ListPreTasks(db database.ReadAccess, rule *model.Rule) ([]*model.Task, error) {
	return listTasks(db, rule, model.ChainPre)
}

func ListPostTasks(db database.ReadAccess, rule *model.Rule) ([]*model.Task, error) {
	return listTasks(db, rule, model.ChainPost)
}

func ListErrorTasks(db database.ReadAccess, rule *model.Rule) ([]*model.Task, error) {
	return listTasks(db, rule, model.ChainError)
}

func setTasks(db *database.DB, rule *model.Rule, chain model.Chain, tasks []*model.Task) error {
	return db.Transaction(func(db *database.Session) error {
		if err := db.DeleteAll(&model.Task{}).Where("rule_id=? AND chain=?",
			rule.ID, chain).Run(); err != nil {
			return err
		}

		for _, task := range tasks {
			if err := db.Insert(task).Run(); err != nil {
				return err
			}
		}

		return nil
	})
}

func SetPreTasks(db *database.DB, rule *model.Rule, tasks []*model.Task) error {
	return setTasks(db, rule, model.ChainPre, tasks)
}

func SetPostTasks(db *database.DB, rule *model.Rule, tasks []*model.Task) error {
	return setTasks(db, rule, model.ChainPost, tasks)
}

func SetErrorTasks(db *database.DB, rule *model.Rule, tasks []*model.Task) error {
	return setTasks(db, rule, model.ChainError, tasks)
}
