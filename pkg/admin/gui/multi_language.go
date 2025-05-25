package gui

import (
	"net/http"
	"time"
)

const hoursDay = 24

func changeLanguage(w http.ResponseWriter, r *http.Request) string {
	userLanguage := r.URL.Query().Get("language")
	const durationCookie = (hoursDay * time.Hour) * 365

	if userLanguage != "" {
		http.SetCookie(w, &http.Cookie{
			Name:    "language",
			Value:   userLanguage,
			Path:    "/",
			Expires: time.Now().Add(durationCookie),
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
	if res[0] == 'e' && res[1] == 'n' {
		return "en"
	}

	if res[0] == 'f' && res[1] == 'r' {
		return "fr"
	}

	if res[0] == 'e' && res[1] == 's' {
		return "es"
	}

	return "en"
}
