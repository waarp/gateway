package rules

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const ruleIDParam = "ruleID"

func GetRule(db database.ReadAccess, r *http.Request) (*model.Rule, error) {
	urlParams := r.URL.Query()
	ruleIDStr := urlParams.Get(ruleIDParam)

	if ruleIDStr == "" {
		return nil, common.NewError(http.StatusBadRequest, "missing ruleID parameter")
	}

	ruleID, parsErr := internal.ParseInt[int64](ruleIDStr)
	if parsErr != nil {
		return nil, common.NewErrorWith(http.StatusBadRequest, "failed to parse ruleID", parsErr)
	}

	var rule model.Rule
	if err := db.Get(&rule, "id=?", ruleID).Run(); err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError, "failed to get rule", err)
	}

	return &rule, nil
}
