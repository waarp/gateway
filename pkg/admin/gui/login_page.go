package gui

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/golang-jwt/jwt/v5"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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

const validTimeToken = 20 * time.Minute

//nolint:gochecknoglobals // secretKey & sessionStore
var (
	secretKey    = CreateSecretKey()
	sessionStore sync.Map
)

//nolint:gochecknoinits // init
func init() {
	const cleaner = 5 * time.Minute
	CleanOldSession(cleaner)
}

func CleanOldSession(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			now := time.Now()

			sessionStore.Range(func(key, value any) bool {
				session, ok := value.(Session)
				if ok && session.Expiration.Before(now) {
					sessionStore.Delete(key)
				}

				return true
			})
		}
	}()
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

func TokenMaxPerUser(user *model.User) {
	var userSessions []Session
	maxPerUser := 5

	sessionStore.Range(func(key, value any) bool {
		session, ok := value.(Session)
		if ok && session.UserID == int(user.ID) {
			userSessions = append(userSessions, session)
		}

		return true
	})
	sort.Slice(userSessions, func(i, j int) bool {
		return userSessions[i].Expiration.Before(userSessions[j].Expiration)
	})

	if len(userSessions) > maxPerUser {
		sessionStore.Delete(userSessions[0].Token)
	}
}

func RefreshExpirationToken(token string) {
	value, ok := sessionStore.Load(token)
	if !ok {
		return
	}

	session, ok := value.(Session)
	if !ok {
		return
	}

	if session.Expiration.After(time.Now()) {
		session.Expiration = time.Now().Add(validTimeToken)
		sessionStore.Store(token, session)
	}
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

var errAuthentication = errors.New("incorrect username or password")

func checkUser(db *database.DB, username, password string) (*model.User, error) {
	user, err := internal.GetUser(db, username)

	passwordHash := ""
	if err == nil {
		passwordHash = user.PasswordHash
	}

	pwd := internal.CheckHash(password, passwordHash)
	if !pwd {
		return nil, errAuthentication
	}

	if err != nil {
		return nil, errAuthentication
	}

	return user, nil
}

func loginPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("login_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert // userLanguage
		var errorMessage string

		if r.Method == http.MethodPost { //nolint:nestif // loginpage
			if err := r.ParseForm(); err != nil {
				logger.Error("Error: %v", err)
				errorMessage = tabTranslated["error"]
			} else {
				username := r.FormValue("username")
				password := r.FormValue("password")

				user, err := checkUser(db, username, password)
				if err != nil {
					logger.Error("Incorrect username or password: %v", err)
					errorMessage = tabTranslated["errorUser"]
				} else {
					token, err := CreateSession(int(user.ID), validTimeToken)
					if err != nil {
						logger.Error("Error creating session: %v", err)
						errorMessage = tabTranslated["errorSession"]
					} else {
						TokenMaxPerUser(user)
						http.SetCookie(w, &http.Cookie{
							Name:     "token",
							Value:    token,
							Path:     "/",
							Expires:  time.Now().Add(validTimeToken),
							Secure:   true,
							HttpOnly: true,
							SameSite: http.SameSiteLaxMode,
						})
						http.Redirect(w, r, "home", http.StatusFound)

						return
					}
				}
			}
		}

		if err := templates.ExecuteTemplate(w, "login_page", map[string]any{
			"tab":      tabTranslated,
			"Error":    errorMessage,
			"language": userLanguage,
		}); err != nil {
			logger.Error("render login_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
