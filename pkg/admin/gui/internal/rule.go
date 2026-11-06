package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetRuleByID(db database.ReadAccess, id int64) (*model.Rule, error) {
	var rule model.Rule

	return &rule, db.Get(&rule, "id=?", id).Run()
}

func ListRules(db database.ReadAccess, orderByCol string, orderByAsc bool,
	limit, offset int,
) ([]*model.Rule, error) {
	var rules model.Rules

	return rules, db.Select(&rules).Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}
