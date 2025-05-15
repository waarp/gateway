package gui

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/golang-jwt/jwt/v5"
)

type Session struct {
	Token      string
	Username   string
	Expiration time.Time
}

func CreateSecretKey() []byte {
	b := 32
	key := make([]byte, b)

	_, err := rand.Read(key)
	if err != nil {
		panic(fmt.Errorf("error: %w", err))
	}

	return []byte(base64.StdEncoding.EncodeToString(key))
}

//nolint:gochecknoglobals // secretKey
var secretKey []byte

//nolint:gochecknoinits // init
func init() {
	secretKey = CreateSecretKey()
}

//nolint:gochecknoglobals // sessionStore
var sessionStore sync.Map

func CreateToken(username string, validTime time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"exp":      time.Now().Add(validTime).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

func CreateSession(username string, validTime time.Duration) (token string, err error) {
	token, err = CreateToken(username, validTime)
	if err != nil {
		return "", err
	}
	session := Session{
		Token:      token,
		Username:   username,
		Expiration: time.Now().Add(validTime),
	}
	sessionStore.Store(token, session)

	return token, nil
}

func ValidateSession(token string) (username string, found bool) {
	value, ok := sessionStore.Load(token)
	if !ok {
		return "", false
	}

	session, ok := value.(Session)
	if !ok {
		return "", false
	}

	if session.Expiration.Before(time.Now()) {
		sessionStore.Delete(token)

		return "", false
	}

	return session.Username, true
}

func DeleteSession(token string) {
	sessionStore.Delete(token)
}

func loginPage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "login_page", map[string]any{"Title": "Se connecter"}); err != nil {
			logger.Error("render login_page: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
