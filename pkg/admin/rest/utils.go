package rest

import (
	"net/http"
	"reflect"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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

func warnHandlerNoDB(h HandlerNoDB, msg string, args ...any) HandlerNoDB {
	return func(logger *log.Logger) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			logger.Warningf(msg, args...)
			h(logger)(w, r)
		}
	}
}
