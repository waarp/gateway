package gui

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/puzpuzpuz/xsync"
)

type Session struct {
	Token      string
	UserID     string
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
var sessionStore xsync.MapOf[string, Session]

func CreateToken(userID string, validTime time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"userID": userID,
			"exp":    time.Now().Add(validTime).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

func CreateSession(userID string, validTime time.Duration) (token string, err error) {
	token, err = CreateToken(userID, validTime)
	if err != nil {
		return "", err
	}
	session := Session{
		Token:      token,
		UserID:     userID,
		Expiration: time.Now().Add(validTime),
	}
	sessionStore.Store(token, session)

	return token, nil
}

func ValidateSession(token string) (userID string, found bool) {
	value, ok := sessionStore.Load(token)
	if !ok {
		return "", false
	}

	if value.Expiration.Before(time.Now()) {
		sessionStore.Delete(token)

		return "", false
	}

	return value.UserID, true
}

func DeleteSession(token string) {
	sessionStore.Delete(token)
}

func checkUser(logger *log.Logger, username, password string) {
	logger.Info("username: %s", username)
	logger.Info("password: %s", password)
}

func loginPage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")
			checkUser(logger, username, password)
		}

		if err := templates.ExecuteTemplate(w, "login_page", map[string]any{"Title": "Se connecter"}); err != nil {
			logger.Error("render login_page: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
