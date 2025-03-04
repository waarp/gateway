package pesit

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	MaxPasswordLength = 8
	maxASCIICharCode  = 127
)

var (
	ErrPasswordTooLong     = errors.New("pesit passwords cannot be longer than 8 characters")
	ErrInvalidPasswordChar = errors.New("pesit passwords can only contain 7-bits ASCII characters")
	ErrLoginTooLong        = errors.New("pesit logins cannot be longer than 8 characters")
	ErrInvalidLoginChar    = errors.New("pesit logins can only contain 7-bits ASCII characters")
	ErrPreConnAuthOnServer = errors.New("pesit pre-connection credentials are not allowed on servers " +
		"(only remote accounts)")
)

const preConnectionAuth = "pesit_pre-connection_auth"

//nolint:gochecknoinits //init is required here
func init() {
	// Internal password
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, Pesit, &pesitBcryptAuthHandler{})
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, PesitTLS, &pesitBcryptAuthHandler{})
	// External password
	authentication.AddExternalCredentialTypeForProtocol(auth.Password, Pesit, &pesitAESAuthHandler{})
	authentication.AddExternalCredentialTypeForProtocol(auth.Password, PesitTLS, &pesitAESAuthHandler{})

	// Pre-connection authentication
	authentication.AddExternalCredentialTypeForProtocol(preConnectionAuth, Pesit, &preConnectionAuthExtHandler{})
	authentication.AddExternalCredentialTypeForProtocol(preConnectionAuth, PesitTLS, &preConnectionAuthExtHandler{})
}

func checkPesitPassword(pswd string) error {
	if len(pswd) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	for _, char := range pswd {
		if char > maxASCIICharCode {
			return ErrInvalidPasswordChar
		}
	}

	return nil
}

func checkPesitLogin(login string) error {
	if len(login) > MaxPasswordLength {
		return ErrLoginTooLong
	}

	for _, char := range login {
		if char > maxASCIICharCode {
			return ErrInvalidLoginChar
		}
	}

	return nil
}

type pesitBcryptAuthHandler struct{ auth.BcryptAuthHandler }

func (p pesitBcryptAuthHandler) Validate(val, val2, protocol, host string, isServer bool) error {
	if err := p.BcryptAuthHandler.Validate(val, val2, protocol, host, isServer); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	return checkPesitPassword(val)
}

type pesitAESAuthHandler struct{ auth.AESPasswordHandler }

func (p pesitAESAuthHandler) Validate(val, val2, protocol, host string, isServer bool) error {
	if err := p.AESPasswordHandler.Validate(val, val2, protocol, host, isServer); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	return checkPesitPassword(val)
}

type preConnectionAuthExtHandler struct{}

func (p preConnectionAuthExtHandler) FromDB(login, cryptPwd string) (string, string, error) {
	plainPwd, err := utils.AESDecrypt(database.GCM, cryptPwd)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt the password: %w", err)
	}

	return login, plainPwd, nil
}

func (p preConnectionAuthExtHandler) ToDB(login, plainPwd string) (string, string, error) {
	cryptPwd, err := utils.AESCrypt(database.GCM, plainPwd)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt the password: %w", err)
	}

	return login, cryptPwd, nil
}

func (p preConnectionAuthExtHandler) CanOnlyHaveOne() bool { return true }
func (p preConnectionAuthExtHandler) Validate(login, pwd, _, _ string, isServer bool) error {
	if isServer {
		return ErrPreConnAuthOnServer
	}

	if err := checkPesitLogin(login); err != nil {
		return err
	}

	return checkPesitPassword(pwd)
}
