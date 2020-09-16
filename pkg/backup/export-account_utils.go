package backup

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

type tableName interface {
	TableName() string
	ElemName() string
}

func accountExists(db *database.Session, account tableName) (bool, error) {
	err := db.Get(account)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
