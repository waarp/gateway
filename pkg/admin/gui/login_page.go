package gui

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type session struct {
	userID         int64
	validityTime   time.Time
	expirationTime time.Time
}

const (
	tokenRefreshDuration  = 20 * time.Minute
	tokenLifetimeDuration = 24 * time.Hour

	tokenCookieName = "token"
)

var (
	ErrAuthentication = errors.New("incorrect username or password")
	ErrTokenUnknown   = errors.New("unknown authentication token")
	ErrTokenInvalid   = errors.New("token is no longer valid")
	ErrTokenExpired   = errors.New("token has expired")
)

//nolint:gochecknoglobals // token storage
var tokens = xsync.NewMap[string, *session]()

func setTokenCookie(r *http.Request, w http.ResponseWriter, token string, ses *session) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  ses.validityTime,
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func addNewToken(r *http.Request, w http.ResponseWriter, userID int64) {
	// invalidate all tokens for this user
	tokens.Range(func(token string, ses *session) bool {
		if ses.userID == userID {
			tokens.Delete(token)
		}

		return true
	})

	// create a new token
	newToken := uuid.New().String()
	newSes := &session{
		userID:         userID,
		validityTime:   time.Now().Add(tokenRefreshDuration),
		expirationTime: time.Now().Add(tokenLifetimeDuration),
	}

	tokens.Store(newToken, newSes)
	setTokenCookie(r, w, newToken, newSes)
}

func refreshToken(r *http.Request, w http.ResponseWriter, token string) {
	ses, ok := tokens.Load(token)
	if !ok {
		return
	}

	if now := time.Now(); now.Before(ses.expirationTime) {
		ses.validityTime = now.Add(tokenRefreshDuration)
	}

	tokens.Store(token, ses)
	setTokenCookie(r, w, token, ses)
}

func invalidateToken(token string) {
	tokens.Delete(token)
}

func validateSession(db database.ReadAccess, token string) (*model.User, error) {
	ses, ok := tokens.Load(token)
	if !ok {
		return nil, ErrTokenUnknown
	}

	now := time.Now()
	if now.After(ses.validityTime) {
		return nil, ErrTokenInvalid
	}

	if now.After(ses.expirationTime) {
		return nil, ErrTokenExpired
	}

	var user model.User
	if err := db.Get(&user, "id=?", ses.userID).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &user, nil
}

func checkUser(db database.ReadAccess, username, password string) (*model.User, error) {
	var user model.User

	err := db.Get(&user, "username=?", username).Owner().Run()
	if err != nil && !database.IsNotFound(err) {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if ok := internal.CheckHash(password, user.PasswordHash); err != nil || !ok {
		return nil, ErrAuthentication
	}

	return &user, nil
}

func loginPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := locale.GetLanguage(r)
		tabTranslated := pageTranslated("login_page", userLanguage)

		if r.Method == http.MethodGet {
			makeLoginPage(w, logger, userLanguage, tabTranslated, "")

			return
		}

		user, errorMsg := authenticateUser(db, logger, r, tabTranslated)
		if errorMsg != "" {
			makeLoginPage(w, logger, userLanguage, tabTranslated, errorMsg)

			return
		}

		addNewToken(r, w, user.ID)

		if redirect := r.URL.Query().Get("redirect"); redirect != "" {
			http.Redirect(w, r, redirect, http.StatusFound)
		} else {
			http.Redirect(w, r, "home", http.StatusFound)
		}
	}
}

func makeLoginPage(w http.ResponseWriter, logger *log.Logger, lang string,
	tabTranslated map[string]string, errorMsg string,
) {
	if tmplErr := loginTemplate.ExecuteTemplate(w, "login_page", map[string]any{
		"tab":      tabTranslated,
		"Error":    errorMsg,
		"language": lang,
	}); tmplErr != nil {
		logger.Errorf("render login_page: %v", tmplErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func authenticateUser(db database.ReadAccess, logger *log.Logger, r *http.Request,
	translation map[string]string,
) (user *model.User, msg string) {
	if err := r.ParseForm(); err != nil {
		logger.Warningf("Failed to parse login form: %v", err)

		return nil, translation["error"]
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, checkErr := checkUser(db, username, password)
	if errors.Is(checkErr, ErrAuthentication) {
		logger.Warning("Incorrect username or password")

		return nil, translation["errorUser"]
	} else if checkErr != nil {
		logger.Errorf("Failed to check user: %v", checkErr)

		return nil, checkErr.Error()
	}

	return user, ""
}
