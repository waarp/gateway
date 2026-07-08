package utils

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

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

// IsHashOf returns whether the given hash is a hash of the given password.
func IsHashOf(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

// ConstantEqual takes a pair of strings and returns whether they are equal or
// not. Comparison is done in constant time for security purposes.
func ConstantEqual(s1, s2 string) bool {
	return subtle.ConstantTimeCompare([]byte(s1), []byte(s2)) == 1
}
