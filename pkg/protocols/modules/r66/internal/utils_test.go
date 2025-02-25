package internal

import (
	"code.waarp.fr/lib/r66"
)

type testAuthHandler func(*r66.Authent) (r66.SessionHandler, error)

func (t testAuthHandler) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) {
	return t(auth)
}

type testSessionHandler func(*r66.Request) (r66.TransferHandler, error)

func (t testSessionHandler) ValidRequest(request *r66.Request) (r66.TransferHandler, error) {
	return t(request)
}
