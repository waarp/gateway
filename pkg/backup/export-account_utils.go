package backup

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

func accountExists(db database.ReadAccess, account database.GetBean, cond string,
	args ...interface{},
) (bool, database.Error) {
	err := db.Get(account, cond, args...).Run()
	if err != nil {
		if database.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
