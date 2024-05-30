// Package authentication defines the handler interfaces which must be implemented
// when adding a new authentication method to the gateway. It also contains the
// list of all the supported authentication methods.
//
// The package also implements a few basic authentication methods.
package authentication

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

const defaultProtocol = ""

// Owner is the interface implemented by all valid credential owner types.
type Owner interface {
	// Host returns the owner's hostname.
	Host() string

	// IsServer returns whether the owner is a server.
	IsServer() bool

	// GetCredCond returns the SQL WHERE clause for selecting credentials belonging
	// to this owner.
	GetCredCond() (string, int64)
}

//nolint:gochecknoglobals //a global var is needed to store the auth handlers
var (
	internalAuthentication = map[string]map[string]InternalAuthHandler{}
	externalAuthentication = map[string]map[string]ExternalAuthHandler{}
)

// AddInternalCredentialType adds the given authentication handler to the list of
// supported forms of internal authentication under the given name.
func AddInternalCredentialType(authType string, handler InternalAuthHandler) {
	AddInternalCredentialTypeForProtocol(authType, defaultProtocol, handler)
}

// AddInternalCredentialTypeForProtocol adds the given authentication handler to
// the list of supported forms of internal authentication under the given name
// AND ONLY for the given protocol.
func AddInternalCredentialTypeForProtocol(authType, protocol string, handler InternalAuthHandler) {
	if internalAuthentication[authType] == nil {
		internalAuthentication[authType] = map[string]InternalAuthHandler{}
	}

	internalAuthentication[authType][protocol] = handler
}

// AddExternalCredentialType adds the given authentication handler to the list of
// supported forms of partner authentication under the given name.
func AddExternalCredentialType(authType string, handler ExternalAuthHandler) {
	AddExternalCredentialTypeForProtocol(authType, defaultProtocol, handler)
}

// AddExternalCredentialTypeForProtocol adds the given authentication handler to
// the list of supported forms of partner authentication under the given name
// AND ONLY for the given protocol.
func AddExternalCredentialTypeForProtocol(authType, protocol string, handler ExternalAuthHandler) {
	if externalAuthentication[authType] == nil {
		externalAuthentication[authType] = map[string]ExternalAuthHandler{}
	}

	externalAuthentication[authType][protocol] = handler
}

// GetInternalAuthHandler returns the InternalAuthHandler associated with the given
// AuthType name. If no handler exists under this name, the function returns nil.
func GetInternalAuthHandler(authType, protocol string) InternalAuthHandler {
	handlers := internalAuthentication[authType]
	if handlers == nil {
		return nil
	}

	if handler := handlers[protocol]; handler != nil {
		return handler
	}

	return handlers[defaultProtocol]
}

// GetExternalAuthMethod returns the ExternalAuthHandler associated with the given
// AuthType name. If no handler exists under this name, the function returns nil.
func GetExternalAuthMethod(authType, protocol string) ExternalAuthHandler {
	handlers := externalAuthentication[authType]
	if handlers == nil {
		return nil
	}

	if handler := handlers[protocol]; handler != nil {
		return handler
	}

	return handlers[defaultProtocol]
}

// Handler is the interface exposing the base functions needed to implement
// a new form of authentication (both internal and external).
type Handler interface {
	// CanOnlyHaveOne states whether the authentication method allows a single
	// owner to possess multiple authentication values of that type.
	CanOnlyHaveOne() bool

	// Validate checks whether the given authentication value is valid. If it's
	// not, the function should return an error. This function is required.
	Validate(value, value2 string, protocol, host string, isServer bool) error
}

// InternalAuthHandler is a struct regrouping the various function necessary to
// perform a distinct form of authentication when connecting to the local
// gateway instance.
type InternalAuthHandler interface {
	Handler

	// Authenticate checks whether the given authentication value 'val' matches
	// the reference value for the given owner. This function is required.
	Authenticate(db database.ReadAccess, owner Owner, val any) (*Result, error)
}

// ExternalAuthHandler is a struct regrouping the various function necessary to
// perform a distinct form of authentication when connecting to a remote partner
// instance.
type ExternalAuthHandler interface {
	Handler
}

// Serializer is an interface which can optionally be implemented by authentication
// handlers when adding a new authentication method, if the authentication value
// needs to be changed when stored in the database.
type Serializer interface {
	// ToDB converts the authentication value into its database format. This
	// function is optional, and is unneeded if the value does not change when
	// stored in the database.
	ToDB(val string, val2 string) (string, string, error)
}

// Deserializer is an interface which can optionally be implemented by authentication
// handlers when adding a new authentication method, if the authentication value
// needs to be changed when retrieved from the database.
type Deserializer interface {
	// FromDB converts an authentication value from its database format into its
	// standard form. This function is optional, and is unneeded if the value
	// does not change when stored in the database.
	FromDB(val string, val2 string) (string, string, error)
}

type Result struct {
	Success bool
	Reason  string
}

func Success() *Result {
	return &Result{Success: true}
}

func Failure(reason string) *Result {
	return &Result{Success: false, Reason: reason}
}
