package rest

import (
	"reflect"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func asNullable[T any](val T) api.Nullable[T] {
	return api.Nullable[T]{Value: val, Valid: !reflect.ValueOf(val).IsZero()}
}

func asNullableBool(b bool) api.Nullable[bool] {
	return api.Nullable[bool]{Value: b, Valid: true}
}

func setIfValid[T any](field *T, value api.Nullable[T]) {
	if value.Valid {
		*field = value.Value
	}
}

func setIfValidList[S ~[]E, E any](field *S, value S) {
	if value != nil {
		*field = value
	}
}
