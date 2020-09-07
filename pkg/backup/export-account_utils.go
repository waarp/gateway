package backup

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

type tableName interface {
	TableName() string
}

func accountExists(db *database.Session, account tableName) (bool, error) {
	err := db.Get(account)
	if err != nil {
		if err == database.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
