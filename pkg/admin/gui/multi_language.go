package gui

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const hoursDay = 24

//nolint:gochecknoglobals // map
var mapLanguage = make(map[string]map[string]map[string]string)

//nolint:gochecknoinits // init
func init() {
	entries, err := webFS.ReadDir("front-end/multi_language")
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		file, err := webFS.ReadFile("front-end/multi_language/" + e.Name())
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
	languageAl := strings.Split(res, ",")

	for _, v := range languageAl {
		if strings.HasPrefix(v, "en") {
			return "en"
		}

		if strings.HasPrefix(v, "fr") {
			return "fr"
		}
	}

	return "en"
}

//nolint:unused // method for later
func tagLanguage(r *http.Request, userLanguage string) language.Tag {
	matcher := language.NewMatcher([]language.Tag{
		language.MustParse("en"),
		language.MustParse("en-US"),
		language.MustParse("en-GB"),
		language.MustParse("en-AU"),

		language.MustParse("fr"),
		language.MustParse("fr-FR"),
		language.MustParse("fr-CA"),
	})

	res := r.Header.Get("Accept-Language")

	tag, _, _ := matcher.Match(language.Make(res))

	if base, _ := tag.Base(); base.String() == userLanguage {
		return tag
	}

	userTag, err := language.Parse(userLanguage)
	if err != nil {
		log.Printf("internal error: %v", err)
	}

	return userTag
}

//nolint:unused // method for later
func translateInt(nb int, r *http.Request, userLanguage string) string {
	msg := message.NewPrinter(tagLanguage(r, userLanguage))

	return msg.Sprintf("%d", nb)
}

//nolint:unused // method for later
func translateFloat(nb float64, r *http.Request, userLanguage string) string {
	msg := message.NewPrinter(tagLanguage(r, userLanguage))

	return msg.Sprintf("%.2f", nb)
}

//nolint:unused // method for later
func translateCurrency(nb float64, r *http.Request, currencyStr, userLanguage string) string {
	tag := tagLanguage(r, userLanguage)

	unit, err := currency.ParseISO(currencyStr)
	if err != nil {
		return message.NewPrinter(tag).Sprintf("%.2f %s", nb, currencyStr)
	}
	res := unit.Amount(nb)
	symbol := currency.Symbol(res)

	return message.NewPrinter(tag).Sprintf("%v", symbol)
}

//nolint:unused // method for later
func translateDateShort(t time.Time, r *http.Request, userLanguage string) string {
	tag := tagLanguage(r, userLanguage).String()
	if tag == "en-US" {
		return t.Format("01-02-2006")
	}

	return t.Format("02/01/2006")
}

//nolint:unused // method for later
func translateTime(t time.Time, r *http.Request, userLanguage string) string {
	tag := tagLanguage(r, userLanguage).String()
	if tag == "en-US" {
		return t.Format("03:04:05 PM")
	}

	return t.Format(time.TimeOnly)
}
