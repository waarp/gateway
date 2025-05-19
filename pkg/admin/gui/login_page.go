package gui

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/golang-jwt/jwt/v5"

    "code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
    "code.waarp.fr/apps/gateway/gateway/pkg/database"
    "code.waarp.fr/apps/gateway/gateway/pkg/model"
    "code.waarp.fr/lib/log"
)

type Session struct {
	Token      string
	UserID     int
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

//nolint:gochecknoglobals // validTimeToken
var validTimeToken = 24 * time.Hour

//nolint:gochecknoglobals // secretKey
var secretKey []byte

//nolint:gochecknoglobals // sessionStore
var sessionStore sync.Map

//nolint:gochecknoinits // init
func init() {
	secretKey = CreateSecretKey()
}

func CreateToken(userID int, validTime time.Duration) (string, error) {
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

func CreateSession(userID int, validTime time.Duration) (token string, err error) {
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

func ValidateSession(token string) (userID int, found bool) {
	value, ok := sessionStore.Load(token)
	if !ok {
		return 0, false
	}

	session, ok := value.(Session)
	if !ok {
		return 0, false
	}

	if session.Expiration.Before(time.Now()) {
		sessionStore.Delete(token)

		return 0, false
	}

	return session.UserID, true
}

func DeleteSession(token string) {
	sessionStore.Delete(token)
}

func checkUser(db *database.DB, username, password string) (*model.User, error) {
	user, err := internal.GetUser(db, username)
	if err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("identifiant invalide: %w", err)
		} else {
			return nil, fmt.Errorf("erreur: %w", err)
		}
	}

	if !internal.CheckHash(password, user.PasswordHash) {
		return nil, fmt.Errorf("mots de passe invalide: %w", err)
	}

	return user, nil
}

func loginPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			err := r.ParseForm()
			if err != nil {
				logger.Error("Erreur: %v", err)
				http.Error(w, "Erreur", http.StatusInternalServerError)

				return
			}
			username := r.FormValue("username")
			password := r.FormValue("password")

			user, err := checkUser(db, username, password)
			if err != nil {
				logger.Error("Erreur d'authentification: %v", err)
				http.Error(w, "Identifiant ou mot de passe invalide", http.StatusUnauthorized)

				return
			}

			token, err := CreateSession(int(user.ID), validTimeToken)
			if err != nil {
				logger.Error("Erreur de la création de la session: %v", err)
				http.Error(w, "Erreur de la création de la session", http.StatusInternalServerError)

				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    token,
				Path:     "/",
				Expires:  time.Now().Add(validTimeToken),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "home", http.StatusFound)

			return
		}

		if err := templates.ExecuteTemplate(w, "login_page", map[string]any{"Title": "Se connecter"}); err != nil {
			logger.Error("render login_page: %v", err)
			http.Error(w, "Erreur interne", http.StatusInternalServerError)
		}
	}
}
