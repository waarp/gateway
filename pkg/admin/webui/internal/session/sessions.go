package session

import (
	"crypto/rand"
	"net/http"
	"time"

	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	SessionCookieName = "session"
	SessionDuration   = 8 * time.Hour
)

type Session struct {
	User      *model.User
	ExpiresAt time.Time
}

type Store struct {
	m *xsync.Map[string, Session]
}

func NewStore() *Store {
	return &Store{m: xsync.NewMap[string, Session]()}
}

func (s *Store) Create(w http.ResponseWriter, user *model.User) {
	token := rand.Text()

	s.m.Store(token, Session{
		User:      user,
		ExpiresAt: time.Now().Add(SessionDuration),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionDuration),
	})
}

func (s *Store) Get(r *http.Request) *model.User {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}

	sess, ok := s.m.Load(cookie.Value)
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil
	}

	return sess.User
}

func (s *Store) Delete(r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return
	}

	s.m.Delete(cookie.Value)
}
