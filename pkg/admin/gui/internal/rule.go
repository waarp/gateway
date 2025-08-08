package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetRule(db database.ReadAccess, name string, isSend bool) (*model.Rule, error) {
	var rule model.Rule

	return &rule, db.Get(&rule, "name=? AND is_send=?", name, isSend).Run()
}

func GetRuleByID(db database.ReadAccess, id int64) (*model.Rule, error) {
	var rule model.Rule

	return &rule, db.Get(&rule, "id=?", id).Run()
}

func GetRulesLike(db *database.DB, prefix string) ([]*model.Rule, error) {
	var rules model.Rules

	return rules, db.Select(&rules).Where("name LIKE ?", prefix+"%").
		OrderBy("name", true).Limit(LimitLike, 0).Run()
}

func ListRules(db database.ReadAccess, orderByCol string, orderByAsc bool,
	limit, offset int,
) ([]*model.Rule, error) {
	var rules model.Rules

	return rules, db.Select(&rules).Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}

func ListRulesByDirection(db database.ReadAccess, orderByCol string, orderByAsc bool,
	limit, offset int, isSend bool,
) ([]*model.Rule, error) {
	var rules model.Rules

	return rules, db.Select(&rules).Limit(limit, offset).
		OrderBy(orderByCol, orderByAsc).Where("is_send=?", isSend).Run()
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
