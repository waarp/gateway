package utils

import (
	"crypto/rand"
	"errors"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"golang.org/x/crypto/bcrypt"
)

// CryptPassword takes a slice of bytes representing a password and returns
// it encrypted using the AES algorithm in Galois Counter Mode with the
// passphrase given in the gateway database configuration.
//
// If the password is already encrypted, it is returned unchanged.
// If the password cannot be encrypted, an error is returned.
func CryptPassword(password []byte) ([]byte, error) {

	// If password is already encrypted, don't encrypt it again.
	if strings.HasPrefix(string(password), "$AES$") {
		return password, nil
	}

	nonce := make([]byte, database.GCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	cipherText := database.GCM.Seal(nonce, nonce, password, nil)
	cipherText = append([]byte("$AES$"), cipherText...)
	return cipherText, nil
}

// DecryptPassword takes a slice representing an AES encrypted password and
// returns it decrypted.
//
// If the password cannot be decrypted, an error is returned.
func DecryptPassword(cipher []byte) ([]byte, error) {
	if !strings.HasPrefix(string(cipher), "$AES$") {
		return cipher, nil
	}
	cryptPassword := cipher[5:]

	nonceSize := database.GCM.NonceSize()
	if len(cryptPassword) < nonceSize {
		return nil, errors.New("the nonce cannot be longer than the text")
	}

	nonce, cipherText := cryptPassword[:nonceSize], cryptPassword[nonceSize:]
	password, err := database.GCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return password, nil
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
		return nil, err
	}
	return hash, nil
}
