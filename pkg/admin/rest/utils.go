package rest

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func asNullableStr(str string) api.Nullable[string] {
	return api.Nullable[string]{Value: str, Valid: str != ""}
}

func asNullableSecret(str types.SecretText) api.Nullable[string] {
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
