package rest

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func asNullableStr(str string) api.Nullable[string] {
	return api.Nullable[string]{Value: str, Valid: str != ""}
}

func asNullableSecret(str database.SecretText) api.Nullable[string] {
	return asNullableStr(string(str))
}

func asNullableBool(b bool) api.Nullable[bool] {
	return api.Nullable[bool]{Value: b, Valid: true}
}

func asNullableTime(t time.Time) api.Nullable[time.Time] {
	return api.Nullable[time.Time]{Value: t, Valid: !t.IsZero()}
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
