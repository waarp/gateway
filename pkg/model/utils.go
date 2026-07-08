package model

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

var errWriteOnView = errors.New("cannot insert/update on a view")

// getCredentials fetch from the database then return the associated Credentials if they exist.
func getCredentials(db database.ReadAccess, owner authentication.Owner,
	authTypes ...string,
) (Credentials, error) {
	var auths Credentials
	query := db.Select(&auths).Where(owner.GetCredCond()).OrderBy("id", true)

	if len(authTypes) > 0 {
		vals := make([]any, len(authTypes))

		for i := range authTypes {
			vals[i] = authTypes[i]
		}

		query.In("type", vals...)
	}

	if err := query.Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the cryptos: %w", err)
	}

	// TODO: get only validate certificates
	return auths, nil
}

func countTrue(b ...bool) int {
	count := 0

	for _, v := range b {
		if v {
			count++
		}
	}

	return count
}
