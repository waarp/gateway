package login

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/session"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/translations"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"github.com/a-h/templ"
)

func ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	login := Login(translations.GetPrinter(r), "")
	templ.Handler(login).ServeHTTP(w, r)
}

type signinHandler struct {
	db       *database.DB
	logger   *log.Logger
	sessions *session.Store
}

func HandleLogin(db *database.DB, logger *log.Logger, sessions *session.Store) *signinHandler {
	return &signinHandler{
		db:       db,
		logger:   logger,
		sessions: sessions,
	}
}

func (s *signinHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	printer := translations.GetPrinter(r)
	if errMsg := s.handleLogin(w, r, printer); errMsg != "" {
		login := Login(printer, errMsg)
		templ.Handler(login).ServeHTTP(w, r)

		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *signinHandler) handleLogin(w http.ResponseWriter, r *http.Request, printer *translations.Printer,
) string {
	username := r.FormValue("username")
	password := r.FormValue("password")

	var user model.User
	if err := s.db.Get(&user, "username=?", username).Run(); err != nil && !database.IsNotFound(err) {
		s.logger.Errorf("Database error: %v", err)

		return printer.Sprintf("Internal database error.")
	}

	if !utils.IsHashOf(user.PasswordHash, password) {
		s.logger.Warningf("Invalid credentials for user %q", username)

		return printer.Sprintf("Invalid credentials.")
	}

	s.sessions.Create(w, &user)

	return ""
}

type logoutHandler struct {
	*signinHandler
}

func HandleLogout(db *database.DB, logger *log.Logger, sessions *session.Store) http.Handler {
	return &logoutHandler{
		signinHandler: HandleLogin(db, logger, sessions),
	}
}

func (l *logoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.sessions.Delete(r)
	http.Redirect(w, r, "/login", http.StatusFound)
}
