package translations

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//go:generate gotext -srclang=en update -out=catalog.go -lang=en,fr code.waarp.fr/apps/gateway/gateway/pkg/admin/webui

//nolint:gochecknoglobals //global vars needed here
var (
	Languages = map[string]string{
		"English":  "en",
		"Français": "fr",
	}
	Options = utils.SortedKeys(Languages)
)

const langCookieName = "language"

// Printer wraps a message.Printer together with the display name of the
// active language, so templates can render both translated text and the
// current language label without needing a second argument.
type Printer struct {
	*message.Printer
	Language string // display name, e.g. "English" or "Français"
}

func getTag(r *http.Request) language.Tag {
	var cookieVal string
	if cookie, err := r.Cookie(langCookieName); err == nil {
		cookieVal = cookie.Value
	}

	return message.MatchLanguage(cookieVal, r.Header.Get("Accept-Language"))
}

func GetPrinter(r *http.Request) *Printer {
	tag := getTag(r)
	code := tag.String()

	displayName := code // fallback to BCP 47 code if not found in map
	for name, c := range Languages {
		if c == code {
			displayName = name
			break
		}
	}

	return &Printer{
		Printer:  message.NewPrinter(tag),
		Language: displayName,
	}
}

func SetLanguage(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")

	http.SetCookie(w, &http.Cookie{
		Name:     langCookieName,
		Value:    lang,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	referer := r.Referer()
	if referer == "" {
		referer = "/"
	}

	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func RenderLink(msg, tagID, url string) string {
	openTag := fmt.Sprintf("[%s]", tagID)
	closeTag := fmt.Sprintf("[/%s]", tagID)

	// Replace markers with actual HTML
	htmlTag := fmt.Sprintf(`<a href="%s" class="underline hover:opacity-80">`, url)

	msg = strings.ReplaceAll(msg, openTag, htmlTag)
	msg = strings.ReplaceAll(msg, closeTag, "</a>")

	return msg
}
