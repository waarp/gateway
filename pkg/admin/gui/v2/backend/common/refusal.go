package common

import (
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// Refused says whether err is a refusal of the model, which a page shows,
// rather than a failure: a validation error, or the refusal to delete an
// object that other objects still use (database.ForeignKeyError).
func Refused(err error) bool {
	return foreignKey(err) != nil || database.IsValidationError(err)
}

// HeldBy says whether err is the refusal to delete an object that objects of
// type T still use: the unfinished transfers, the folder watchers. The model
// names the first kind that holds the object back, the transfers before the
// folder watchers.
func HeldBy[T database.FKElement](err error) bool {
	fk := foreignKey(err)
	if fk == nil {
		return false
	}

	for _, elem := range fk.Elems {
		if _, of := elem.(T); of {
			return true
		}
	}

	return false
}

// foreignKey is the refusal to delete an object that other objects still
// use, nil for any other error.
func foreignKey(err error) *database.ForeignKeyError {
	fk, _ := errors.AsType[*database.ForeignKeyError](err)

	return fk
}
