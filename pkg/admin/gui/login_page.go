package gui

import (
	"net/http"
	"github.com/golang-jwt/jwt/v5"
 	"sync"
	"time"
	
	"code.waarp.fr/lib/log"
)

type Session struct {
    Token     string
    Username    string
    Expiration time.Time
}

var secretKey = []byte("Sh2Hdj1SdMknb8sdDzHd7")
var sessionStore sync.Map

func CreateToken(username string, valid_time time.Duration) (string, error) {
   token := jwt.NewWithClaims(jwt.SigningMethodHS256, 
        jwt.MapClaims{
		"username": username,
        "exp": time.Now().Add(valid_time).Unix(),
    })

    tokenString, err := token.SignedString(secretKey)
    if err != nil {
    	return "", err
    }

	return tokenString, nil
}

func CreateSession(username string, valid_time time.Duration) (token string, err error) {
    token, err = CreateToken(username, valid_time)
    if err != nil {
        return "", err
    }
    session := Session{
        Token:     token,
        Username:    username,
        Expiration: time.Now().Add(valid_time),
    }
    sessionStore.Store(token, session)
    return token, nil
}

func ValidateSession(token string) (username string, found bool) {
    value, ok := sessionStore.Load(token)
    if !ok {
        return "", false
    }
    session := value.(Session)
    if session.Expiration.Before(time.Now()) {
        sessionStore.Delete(token)
        return "", false
    }
    return session.Username, true
}

func DeleteSession(token string) {
    sessionStore.Delete(token)
}

func login_page(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "login_page", map[string]any{"Title": "Se connecter"}); err != nil {
			logger.Error("render login_page: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
