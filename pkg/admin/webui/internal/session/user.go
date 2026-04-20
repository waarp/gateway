package session

import (
	"context"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type userKeyType string

const userKey userKeyType = "user"

func WithUser(r *http.Request, user *model.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userKey, user))
}

func GetUser(r *http.Request) *model.User {
	user := r.Context().Value(userKey).(*model.User)
	if user == nil {
		return &model.User{ID: 1, Username: "Dev", Permissions: model.PermAll}
	}

	return user
}
