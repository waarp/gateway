// Package home serves the home page of the web interface, where the user
// lands after signing in: one card per section of the sidebar, one link card
// per screen the user may read, with a sentence saying what the screen holds.
// It reads nothing: the rights come from the session. It also serves the
// "About" page, the fallback of the dialog of the account menu without
// JavaScript.
package home

import (
	"net/http"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The sentence under a screen: "home.desc." + the name of the screen in
	// the screen map ("servers").
	descKeyPrefix = "home.desc."
)

type section struct {
	Key   string
	Title string
	Cards []gui.LinkCardProps // the screens the user may open: icon, name, sentence of the glossary
}

type pageData struct {
	Sections []section
}

// sections builds the cards of a user from the screen map: the same screens
// and rights as the sidebar.
func sections(translate gui.Translator, perms model.PermsMask) []section {
	screens := common.Screens(perms)
	out := make([]section, 0, len(screens))

	for _, s := range screens {
		cards := make([]gui.LinkCardProps, 0, len(s.Screens))

		for _, screen := range s.Screens {
			cards = append(cards, gui.LinkCardProps{
				Icon:  screen.Icon,
				Label: translate(screen.LabelKey),
				Text:  translate(descKeyPrefix + screen.Key),
				URL:   screen.URL,
			})
		}

		out = append(out, section{Key: s.Key, Title: translate(s.TitleKey), Cards: cards})
	}

	return out
}

// Page serves GET /webui/home.
func Page(logger *log.Logger) http.Handler {
	shell := common.NewShell()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		translate := shell.Translator(r)

		// No breadcrumb: it would only repeat the title (Driss, 2026-10-19).
		shell.Render(w, logger, r, common.Screen{Title: translate("home.title")},
			page(pageData{Sections: sections(translate, common.GetUser(r).Permissions)}))
	})
}

// About serves GET /webui/about: the content of the "About" dialog as a
// page, for a browser without JavaScript or a link to keep.
func About(logger *log.Logger) http.Handler {
	shell := common.NewShell()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		translate := shell.Translator(r)

		shell.Render(w, logger, r, common.Screen{
			Title: translate("about.title", "product", constants.AppName),
			Crumbs: []gui.Crumb{
				{Label: translate("home.title"), URL: common.WebuiURL("/home")},
				{Label: translate("about.title", "product", constants.AppName)},
			},
		}, aboutPage(constants.AppName, common.WebuiURL("/home"), common.About(translate)))
	})
}
