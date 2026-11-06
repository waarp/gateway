package gui

import (
	"net/http"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
)

const (
	hoursDay = 24
	English  = "en"
	French   = "fr"
)

func changeLanguage(w http.ResponseWriter, r *http.Request) string {
	userLanguage := r.URL.Query().Get(constants.LanguageParam)
	const durationCookie = (hoursDay * time.Hour) * 365

	secure := r.TLS != nil

	if userLanguage != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "language",
			Value:    userLanguage,
			Path:     "/",
			Expires:  time.Now().Add(durationCookie),
			Secure:   secure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		return userLanguage
	}

	if lang, err := r.Cookie("language"); err == nil && lang.Value != "" {
		return lang.Value
	}

	return detectLanguage(r)
}

func detectLanguage(r *http.Request) string {
	res := r.Header.Get("Accept-Language")

	for v := range strings.SplitSeq(res, ",") {
		if strings.HasPrefix(v, English) {
			return English
		}

		if strings.HasPrefix(v, French) {
			return French
		}
	}

	return English
}
