package authtest

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

type dummyAuthHandler struct{}

func (dummyAuthHandler) CanOnlyHaveOne() bool                     { return false }
func (dummyAuthHandler) Validate(_, _, _, _ string, _ bool) error { return nil }
func (dummyAuthHandler) Authenticate(database.ReadAccess, authentication.Owner, any) (*authentication.Result, error) {
	return authentication.Success(), nil
}

func AddDummyAuthHandler(name, protocol string) {
	authentication.AddInternalCredentialTypeForProtocol(name, protocol, dummyAuthHandler{})
	authentication.AddExternalCredentialTypeForProtocol(name, protocol, dummyAuthHandler{})
}
