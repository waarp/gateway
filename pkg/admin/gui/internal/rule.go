package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetRule(db database.ReadAccess, name string, isSend bool) (*model.Rule, error) {
	var rule model.Rule

	return &rule, db.Get(&rule, "name=? AND is_send=?", name).Run()
}

func ListRules(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
) ([]*model.Rule, error) {
	var rules model.Rules

	return rules, db.Select(&rules).Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}

func InsertRule(db database.Access, rule *model.Rule) error {
	return db.Insert(rule).Run()
}

func UpdateRule(db database.Access, rule *model.Rule) error {
	return db.Update(rule).Run()
}

func DeleteRule(db database.Access, rule *model.Rule) error {
	return db.Delete(rule).Run()
}
