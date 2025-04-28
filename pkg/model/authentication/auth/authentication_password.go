// Package auth regroups handlers for common types of authentications shared by
// multiple protocols.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const Password = "password"

type ProtocolPasswordHandler struct{}

//nolint:gochecknoinits //init is used by design
func init() {
	authentication.AddInternalCredentialType(Password, &BcryptAuthHandler{})
	authentication.AddExternalCredentialType(Password, &AESPasswordHandler{})
}

var ErrEmptyPassword = errors.New("password input is empty")

type BcryptAuthHandler struct{}

func (*BcryptAuthHandler) CanOnlyHaveOne() bool { return true }

func (*BcryptAuthHandler) ToDB(val, _ string) (string, string, error) {
	hashed, err := utils.HashPassword(database.BcryptRounds, val)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash the password: %w", err)
	}

	return hashed, "", nil
}

func (*BcryptAuthHandler) Validate(value, _, _, _ string, _ bool) error {
	if _, err := bcrypt.Cost([]byte(value)); err == nil {
		return nil // password is already hashed
	}

	// TODO add more verifications (min length, character variety...)
	if value == "" {
		return ErrEmptyPassword
	}

	return nil
}

func (*BcryptAuthHandler) Authenticate(db database.ReadAccess,
	owner authentication.Owner, val any,
) (*authentication.Result, error) {
	doVerify := func(value []byte) (*authentication.Result, error) {
		var pswd model.Credential
		if err := db.Get(&pswd, "type=?", Password).And(owner.GetCredCond()).
			Run(); err != nil && !database.IsNotFound(err) {
			return nil, fmt.Errorf("failed to retrieve the reference password hash: %w", err)
		}

		reference := []byte(pswd.Value)

		if bcrypt.CompareHashAndPassword(reference, value) != nil {
			return authentication.Failure("incorrect password"), nil
		}

		return authentication.Success(), nil
	}

	switch value := val.(type) {
	case string:
		return doVerify([]byte(value))
	case []byte:
		return doVerify(value)
	default:
		//nolint:goerr113 //this is a base error
		return nil, fmt.Errorf(`unknown bcrypt hash type "%T"`, value)
	}
}

type AESPasswordHandler struct{}

func (*AESPasswordHandler) CanOnlyHaveOne() bool { return true }

func (*AESPasswordHandler) ToDB(val, _ string) (string, string, error) {
	encrypted, err := utils.AESCrypt(database.GCM, val)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt the password: %w", err)
	}

	return encrypted, "", nil
}

func (*AESPasswordHandler) FromDB(val, _ string) (string, string, error) {
	plain, err := utils.AESDecrypt(database.GCM, val)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt the password: %w", err)
	}

	return plain, "", nil
}

func (*AESPasswordHandler) Validate(value, _, _, _ string, _ bool) error {
	// TODO add more verifications (min length, character variety...)
	if value == "" {
		return ErrEmptyPassword
	}

	return nil
}
