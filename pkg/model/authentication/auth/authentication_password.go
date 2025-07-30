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

func (*BcryptAuthHandler) ToDB(plain, _ string) (hashed, _ string, err error) {
	if hashed, err = utils.HashPassword(database.BcryptRounds, plain); err != nil {
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

func (b *BcryptAuthHandler) Authenticate(db database.ReadAccess,
	owner authentication.Owner, val any,
) (*authentication.Result, error) {
	return b.AuthenticateType(db, owner, val, Password)
}

func (*BcryptAuthHandler) AuthenticateType(db database.ReadAccess,
	owner authentication.Owner, val any, authType string,
) (*authentication.Result, error) {
	doVerify := func(value []byte) (*authentication.Result, error) {
		var pswd model.Credential
		if err := db.Get(&pswd, "type=?", authType).And(owner.GetCredCond()).
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
		//nolint:err113 //this is a base error
		return nil, fmt.Errorf(`unknown bcrypt hash type "%T"`, value)
	}
}

type AESPasswordHandler struct{}

func (*AESPasswordHandler) CanOnlyHaveOne() bool { return true }

func (*AESPasswordHandler) ToDB(plainPwd, _ string) (encryptedPwd, _ string, err error) {
	if encryptedPwd, err = utils.AESCrypt(database.GCM, plainPwd); err != nil {
		return "", "", fmt.Errorf("failed to encrypt the password: %w", err)
	}

	return encryptedPwd, "", nil
}

func (*AESPasswordHandler) FromDB(encryptedPwd, _ string) (plainPwd, _ string, err error) {
	if plainPwd, err = utils.AESDecrypt(database.GCM, encryptedPwd); err != nil {
		return "", "", fmt.Errorf("failed to decrypt the password: %w", err)
	}

	return plainPwd, "", nil
}

func (*AESPasswordHandler) Validate(pwd, _, _, _ string, _ bool) error {
	// TODO add more verifications (min length, character variety...)
	if pwd == "" {
		return ErrEmptyPassword
	}

	return nil
}
