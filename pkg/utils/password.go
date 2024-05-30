package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

var errNonceTooLong = errors.New("the nonce cannot be longer than the text")

// AESCrypt takes a slice of bytes and returns it encrypted using the AES
// algorithm in Galois Counter Mode using the passphrase given in the gateway
// database configuration.
//
// If the slice is already encrypted, it is returned unchanged.
// If the slice cannot be encrypted, an error is returned.
func AESCrypt(gcm cipher.AEAD, password string) (string, error) {
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("cannot get random bytes: %w", err)
	}

	cipherBytes := gcm.Seal(nonce, nonce, []byte(password), nil)
	cipherText := base64.StdEncoding.EncodeToString(cipherBytes)

	return cipherText, nil
}

// AESDecrypt takes a slice representing an AES encrypted text and
// returns it decrypted.
//
// If the slice cannot be decrypted, an error is returned.
func AESDecrypt(gcm cipher.AEAD, cipherStr string) (string, error) {
	cryptPassword, err := base64.StdEncoding.DecodeString(cipherStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted password string: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(cryptPassword) < nonceSize {
		return "", errNonceTooLong
	}

	nonce, cipherText := cryptPassword[:nonceSize], cryptPassword[nonceSize:]

	password, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("cannot decrypt password: %w", err)
	}

	return string(password), nil
}

// IsHash returns whether the given string is a bcrypt hash or not.
func IsHash(password string) bool {
	_, isHashed := bcrypt.Cost([]byte(password))

	return isHashed == nil
}

// HashPassword takes a slice of bytes representing a password and returns it
// hashed using the bcrypt hashing algorithm.
//
// If the password is already hashed, the hash is returned unchanged.
// If the password cannot be hashed, an error is returned.
func HashPassword(bcryptRounds int, password string) (string, error) {
	if password == "" {
		return "", nil
	}

	// If password is already hashed, don't encrypt it again.
	if IsHash(password) {
		return password, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptRounds)
	if err != nil {
		return "", fmt.Errorf("cannot hash password: %w", err)
	}

	return string(hash), nil
}

// ConstantEqual takes a pair of strings and returns whether they are equal or
// not. Comparison is done in constant time for security purposes.
func ConstantEqual(s1, s2 string) bool {
	return subtle.ConstantTimeCompare([]byte(s1), []byte(s2)) == 1
}
