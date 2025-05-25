package gui

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

const hoursDay = 24

//nolint:gochecknoglobals // map
var mapLanguage = make(map[string]map[string]map[string]string)

//nolint:gochecknoinits // init
func init() {
	entries, err := webFS.ReadDir("front_end/multi_language")
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		file, err := webFS.ReadFile("front_end/multi_language/" + e.Name())
		if err != nil {
			log.Fatal(err)
		}

		page := make(map[string]map[string]string)
		if err := json.Unmarshal(file, &page); err != nil {
			log.Fatal(err)
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		mapLanguage[id] = page
	}
}

func pageTranslated(id, userLanguage string) map[string]string {
	tabTranslated := make(map[string]string)

	for key, translation := range mapLanguage[id] {
		tabTranslated[key] = translation[userLanguage]
	}

	return tabTranslated
}

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
