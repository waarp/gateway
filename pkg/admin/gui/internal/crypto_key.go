package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func ListCryptoKeys(db database.ReadAccess, orderByCol string, orderByAsc bool,
	limit, offset int, keyTypes ...string,
) ([]*model.CryptoKey, error) {
	var keys model.CryptoKeys

	return keys, db.Select(&keys).Owner().OrderBy(orderByCol, orderByAsc).
		Limit(limit, offset).In("type", utils.AsAny(keyTypes)...).Run()
}
