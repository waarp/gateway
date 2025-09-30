package internal

import (
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func HashPassword(password string) (string, error) {
	return utils.HashPassword(database.BcryptRounds, password)
}

func CheckHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
