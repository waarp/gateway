package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

var errNonceTooLong = errors.New("the nonce cannot be longer than the text")

// AESCrypt takes a slice of bytes and returns it encrypted using the AES
// algorithm in Galois Counter Mode using the passphrase given in the gateway
// database configuration.
//
// If the slice is already encrypted, it is returned unchanged.
// If the slice cannot be encrypted, an error is returned.
func AESCrypt(password string) (string, error) {
	// If password is already encrypted, don't encrypt it again.
	if strings.HasPrefix(password, "$AES$") {
		return password, nil
	}

	nonce := make([]byte, database.GCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("cannot get random bytes: %w", err)
	}

	cipher := database.GCM.Seal(nonce, nonce, []byte(password), nil)
	cipherText := "$AES$" + base64.StdEncoding.EncodeToString(cipher)

	return cipherText, nil
}

// AESDecrypt takes a slice representing an AES encrypted text and
// returns it decrypted.
//
// If the slice cannot be decrypted, an error is returned.
func AESDecrypt(cipher string) (string, error) {
	if !strings.HasPrefix(cipher, "$AES$") {
		return cipher, nil
	}

	cryptPassword, err := base64.StdEncoding.DecodeString(cipher[5:])
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted password string: %w", err)
	}

	nonceSize := database.GCM.NonceSize()
	if len(cryptPassword) < nonceSize {
		return "", errNonceTooLong
	}

	nonce, cipherText := cryptPassword[:nonceSize], cryptPassword[nonceSize:]

	password, err := database.GCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("cannot decrypt password: %w", err)
	}

	return string(password), nil
}

// HashPassword takes a slice of bytes representing a password and returns it
// hashed using the bcrypt hashing algorithm.
//
// If the password is already hashed, the hash is returned unchanged.
// If the password cannot be hashed, an error is returned.
func HashPassword(password []byte) ([]byte, error) {
	// If password is already hashed, don't encrypt it again.
	if _, isHashed := bcrypt.Cost(password); isHashed == nil {
		return password, nil
	}

	hash, err := bcrypt.GenerateFromPassword(password, database.BcryptRounds)
	if err != nil {
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}

	return hash, nil
}
