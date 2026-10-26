package gui

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/login"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/throttle"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// throttleScope identifies the WebUI login in the authentication limiter.
const throttleScope = "webui"

type session struct {
	userID         int64
	validityTime   time.Time
	expirationTime time.Time
}

const (
	tokenRefreshDuration  = 20 * time.Minute
	tokenLifetimeDuration = 24 * time.Hour

	tokenCookieName = "token"

	// The page asked for before the sign-in, as the authentication middleware
	// sets it; the sign-in leads there.
	redirectParam = "redirect"
	homePath      = "home"
	loginPath     = "login"
	logoutPath    = "logout"
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
	if err := db.Get(&user, "id=?", ses.userID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &user, nil
}

func checkUser(db database.ReadAccess, username, password string) (*model.User, error) {
	var user model.User

	err := db.Get(&user, "username=?", username).Run()
	if err != nil && !database.IsNotFound(err) {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if ok := utils.IsHashOf(user.PasswordHash, password); err != nil || !ok {
		return nil, ErrAuthentication
	}

	return &user, nil
}

// loginPage serves the sign-in page (#787): GET shows the form, with the
// reason the user is there when the address says one; POST checks the
// credentials and opens the session, or shows the form again with the
// refusal (422), the username kept and the password never. The page is
// rendered by the plain page of the kit, in the language of the request.
func loginPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	shell := common.NewShell()

	render := func(w http.ResponseWriter, r *http.Request, data login.Data, status int) {
		translate := shell.Translator(r)
		shell.RenderPlain(w, logger, r, login.Title(translate), status, nil, login.Page(data))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		translate := shell.Translator(r)

		if r.Method == http.MethodGet {
			render(w, r, login.Data{Info: login.Info(translate, r.URL.Query().Get(login.ReasonParam))}, 0)

			return
		}

		user, err := authenticateUser(db, logger, r)
		if err != nil {
			render(w, r, login.Data{
				Username: r.FormValue("username"),
				Error:    login.Refusal(translate, errors.Is(err, ErrAuthentication)),
			}, http.StatusUnprocessableEntity)

			return
		}

		addNewToken(r, w, user.ID)
		http.Redirect(w, r, afterLogin(r.URL.Query().Get(redirectParam)), http.StatusSeeOther)
	}
}

// afterLogin is where the sign-in leads: the page asked for when it is a page
// of the web interface (a path under the prefix, neither the sign-in nor the
// sign-out), the home page otherwise. The address is given by the browser:
// nothing outside the interface is followed.
func afterLogin(target string) string {
	if !strings.HasPrefix(target, Prefix+"/") {
		return homePath
	}

	parsed, err := url.Parse(target)
	if err != nil || parsed.Host != "" || parsed.Scheme != "" {
		return homePath
	}

	switch strings.Trim(strings.TrimPrefix(parsed.Path, Prefix+"/"), "/") {
	case "", loginPath, logoutPath:
		return homePath
	}

	return target
}

// authenticateUser reads the credentials of the form and checks them, within
// the authentication limiter: a locked account is refused like a wrong
// password, without any delay or hint.
func authenticateUser(db *database.DB, logger *log.Logger, r *http.Request) (*model.User, error) {
	if err := r.ParseForm(); err != nil {
		logger.Warningf("Failed to parse login form: %v", err)

		return nil, ErrAuthentication
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	attempt := throttle.Default().Attempt(&db.GetConfig().Security, throttleScope,
		username, r.RemoteAddr)

	if attempt.Locked() {
		logger.Warningf("Login of user %q from %q refused: too many failed attempts",
			username, r.RemoteAddr)

		return nil, ErrAuthentication
	}

	user, checkErr := checkUser(db, username, password)
	if errors.Is(checkErr, ErrAuthentication) {
		logger.Warning("Incorrect username or password")

		if attempt.Fail() {
			logger.Warningf("Too many failed logins for user %q from %q, new attempts are temporarily refused",
				username, r.RemoteAddr)
		}

		return nil, ErrAuthentication
	} else if checkErr != nil {
		logger.Errorf("Failed to check user: %v", checkErr)

		return nil, checkErr
	}

	attempt.Succeed()

	return user, nil
}
