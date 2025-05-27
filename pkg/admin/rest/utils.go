package rest

import (
	"reflect"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func asNullable[T any](val T) api.Nullable[T] {
	return api.Nullable[T]{Value: val, Valid: !reflect.ValueOf(val).IsZero()}
}

func asNullableSecret(str database.SecretText) api.Nullable[string] {
	return asNullable(string(str))
}

func asNullableBool(b bool) api.Nullable[bool] {
	return api.Nullable[bool]{Value: b, Valid: true}
}

func setIfValid[T any](field *T, value api.Nullable[T]) {
	if value.Valid {
		*field = value.Value
	}
}

func setIfValidSecret(field *database.SecretText, value api.Nullable[string]) {
	if value.Valid {
		*field = database.SecretText(value.Value)
	}
}
