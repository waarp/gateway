package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func accountExists(db database.ReadAccess, account database.GetBean, cond string,
	args ...interface{},
) (bool, error) {
	if err := db.Get(account, cond, args...).Run(); err != nil {
		if database.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to retrieve account: %w", err)
	}

	return true, nil
}
