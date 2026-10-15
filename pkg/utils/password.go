package utils

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const BcryptMinCost = bcrypt.MinCost

//nolint:gochecknoglobals //needed here to change cost for tests
var BcryptRounds = 12

func ChangeBcryptRounds(cost int) (reset func()) {
	oldCost := BcryptRounds
	BcryptRounds = cost

	return func() { BcryptRounds = oldCost }
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
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(BcryptRounds, password)
}

func HashPasswordWithCost(bcryptRounds int, password string) (string, error) {
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
	return CheckPassword(BcryptRounds, hash, password)
}

// IsBHashOf returns whether the given hash is a hash of the given password.
func IsBHashOf(hash, password []byte) bool {
	return CheckPassword(BcryptRounds, string(hash), string(password))
}

// dummyHash returns a bcrypt hash of the given cost, which matches no password
// a user can send. It is built once per cost.
func dummyHash(bcryptRounds int) string {
	const bcryptHashLength = 53

	return fmt.Sprintf("$2a$%d$%s", bcryptRounds, strings.Repeat("0", bcryptHashLength))
}

// CheckPassword returns whether the given password matches the given bcrypt
// hash. When there is no reference hash (unknown account, or account without
// password), the password is still compared with a dummy hash of the given
// cost before returning false, so that the time taken does not reveal whether
// the account exists.
func CheckPassword(bcryptRounds int, hash, password string) bool {
	if hash == "" {
		hash = dummyHash(bcryptRounds)
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ConstantEqual takes a pair of strings and returns whether they are equal or
// not. Comparison is done in constant time for security purposes.
func ConstantEqual(s1, s2 string) bool {
	return subtle.ConstantTimeCompare([]byte(s1), []byte(s2)) == 1
}
