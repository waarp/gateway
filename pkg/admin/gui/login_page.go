package gui

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Session struct {
	Token      string
	UserID     int64
	Expiration time.Time
	Filters    filtersCookieMap
}

const (
	validTimeToken  = 10 * time.Second
	tokenCookieName = "token"
	secretKeyLen    = 32
)

var ErrTokenInvalid = errors.New("token has been invalidated")

//nolint:gochecknoglobals // secret key & invalid tokens
var (
	secretKey     = CreateSecretKey()
	invalidTokens = xsync.NewMap[string, time.Time]()
)

func clearInvalidTokens() {
	now := time.Now()
	invalidTokens.Range(func(token string, exp time.Time) bool {
		if exp.Before(now) {
			invalidTokens.Delete(token)
		}

		return true
	})
}

func isTokenIvalidated(candidate string) bool {
	clearInvalidTokens()

	if _, ok := invalidTokens.Load(candidate); ok {
		return false
	}

	return true
}

func invalidateToken(tokenString string) {
	if _, ok := invalidTokens.Load(tokenString); ok {
		return // already invalidated
	}

	token, err := parseToken(tokenString)
	if err != nil {
		return // invalid token
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		exp = &jwt.NumericDate{Time: time.Now().Add(validTimeToken)}
	}

	invalidTokens.Store(tokenString, exp.Time)
}

func CreateSecretKey() []byte {
	key := make([]byte, secretKeyLen)
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Errorf("failed to make JWT secret key: %w", err))
	}

	return key
}

func CreateToken(userID int64) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss": conf.GlobalConfig.GatewayName,
			"sub": utils.FormatInt(userID),
			"iat": now.Unix(),
			"nbf": now.Unix(),
			"exp": now.Add(validTimeToken).Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

func RefreshExpirationToken(logger *log.Logger, w http.ResponseWriter,
	r *http.Request, userID int64,
) {
	token, err := CreateToken(userID)
	if err != nil {
		logger.Warningf("Failed to create token: %v", err)

		return
	}

	secure := r.TLS != nil

	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(validTimeToken),
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func parseToken(tokenString string) (*jwt.Token, error) {
	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(conf.GlobalConfig.GatewayName),
	)

	token, err := parser.Parse(tokenString, func(*jwt.Token) (any, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if isTokenIvalidated(tokenString) {
		return nil, ErrTokenInvalid
	}

	return token, nil
}

//nolint:err113 // these are base errors
func ValidateSession(db database.ReadAccess, tokenString string) (*model.User, error) {
	token, err := parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	subject, err := token.Claims.GetSubject()
	if err != nil || subject == "" {
		return nil, errors.New(`missing "sub" in JWT`)
	}

	userID, err := utils.ParseInt[int64](subject)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subject %q in JWT: %w", subject, err)
	}

	user, err := internal.GetUserByID(db, userID)
	if database.IsNotFound(err) {
		return nil, errors.New("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user with ID %d: %w", userID, err)
	}

	return user, nil
}

var errAuthentication = errors.New("incorrect username or password")

func checkUser(db database.ReadAccess, username, password string) (*model.User, error) {
	var user model.User

	err := db.Get(&user, "username=?", username).Owner().Run()
	if err != nil && !database.IsNotFound(err) {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if ok := internal.CheckHash(password, user.PasswordHash); err != nil || !ok {
		return nil, errAuthentication
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

		RefreshExpirationToken(logger, w, r, user.ID)

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
	if errors.Is(checkErr, errAuthentication) {
		logger.Warning("Incorrect username or password")

		return nil, translation["errorUser"]
	} else if checkErr != nil {
		logger.Errorf("Failed to check user: %v", checkErr)

		return nil, checkErr.Error()
	}

	return user, ""
}
