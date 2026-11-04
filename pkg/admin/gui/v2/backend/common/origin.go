package common

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
)

const (
	// OriginParam names, in the address of a page, the page the user comes
	// from (#840): the address of that page as displayed, under the web
	// interface prefix. The shell renders the return link from it; the list
	// state carries it so that the return chains (liststate). Not "from":
	// the transfer monitoring names the start of its period so.
	OriginParam = "origin"
	// MaxOriginLength bounds the origin an address carries.
	MaxOriginLength = 2048
)

// Origin is a screen a link may come from: its route under the web interface
// prefix, as the router registers it, and the key of the label of the return
// link ("Back to the dashboard"). The forms are no origin: the user leaves
// them by their own buttons.
type Origin struct{ Path, Label string }

// Origins are the screens that may be an origin (#840): the lists, the
// sheets and their tabs, the operations screens. A test of the router checks
// that each one is a GET route of the interface.
//
//nolint:gochecknoglobals // reference data of the screens
var Origins = []Origin{
	{"/dashboard", "origin.dashboard"},
	{"/home", "origin.home"},
	{"/about", "origin.about"},
	{"/services", "origin.services"},
	{"/transfers", "origin.transfers"},
	{"/transfers/{id:[0-9]+}", "origin.transfer"},
	{"/servers", "origin.servers"},
	{"/servers/{id:[0-9]+}", "origin.server"},
	{"/servers/{id:[0-9]+}/auth", "origin.server"},
	{"/servers/{id:[0-9]+}/accounts", "origin.server"},
	{"/servers/{id:[0-9]+}/accounts/{aid:[0-9]+}", "origin.account"},
	{"/servers/{id:[0-9]+}/accounts/{aid:[0-9]+}/auth", "origin.account"},
	{"/servers/{id:[0-9]+}/auth/{cid:[0-9]+}", "origin.credential"},
	{"/servers/{id:[0-9]+}/accounts/{aid:[0-9]+}/auth/{cid:[0-9]+}", "origin.credential"},
	{"/clients", "origin.clients"},
	{"/clients/{id:[0-9]+}", "origin.client"},
	{"/partners", "origin.partners"},
	{"/partners/{id:[0-9]+}", "origin.partner"},
	{"/partners/{id:[0-9]+}/auth", "origin.partner"},
	{"/partners/{id:[0-9]+}/accounts", "origin.partner"},
	{"/partners/{id:[0-9]+}/accounts/{aid:[0-9]+}", "origin.account"},
	{"/partners/{id:[0-9]+}/accounts/{aid:[0-9]+}/auth", "origin.account"},
	{"/partners/{id:[0-9]+}/auth/{cid:[0-9]+}", "origin.credential"},
	{"/partners/{id:[0-9]+}/accounts/{aid:[0-9]+}/auth/{cid:[0-9]+}", "origin.credential"},
	{"/rules", "origin.rules"},
	{"/rules/{id:[0-9]+}", "origin.rule"},
	{"/rules/{id:[0-9]+}/tasks", "origin.rule"},
	{"/rules/{id:[0-9]+}/access", "origin.rule"},
	{"/watchers", "origin.watchers"},
	{"/watchers/{id:[0-9]+}", "origin.watcher"},
	{"/keys", "origin.keys"},
	{"/keys/{id:[0-9]+}", "origin.key"},
	{"/smtp", "origin.smtp"},
	{"/smtp/{id:[0-9]+}", "origin.smtp_credential"},
	{"/email-templates", "origin.email_templates"},
	{"/email-templates/{id:[0-9]+}", "origin.email_template"},
	{"/users", "origin.users"},
	{"/users/{id:[0-9]+}", "origin.user"},
	{"/authorities", "origin.authorities"},
	{"/authorities/{id:[0-9]+}", "origin.authority"},
	{"/clouds", "origin.clouds"},
	{"/clouds/{id:[0-9]+}", "origin.cloud"},
	{"/snmp", "origin.snmp"},
	{"/snmp/monitors/{id:[0-9]+}", "origin.monitor"},
	{"/overrides", "origin.overrides"},
	{"/config", "origin.config"},
}

// originRouter matches an address against the screens that may be an
// origin, each route named by the key of its label.
//
//nolint:gochecknoglobals // built once from Origins
var originRouter = sync.OnceValue(func() *mux.Router {
	router := mux.NewRouter()
	for _, o := range Origins {
		router.Path(o.Path).Methods(http.MethodGet).Name(o.Label)
	}

	return router
})

// ReturnOf is the return link of a page (#840): to the page of origin the
// request carries (?origin=) when it names a screen that may be an origin,
// else the default the handler gives (the list of a sheet), else nil. The
// label comes from the screen matched, never from the address.
func ReturnOf(r *http.Request, translate gui.Translator, fallback *gui.Link) *gui.Link {
	if label, target, ok := OriginOf(r.URL.Query().Get(OriginParam)); ok {
		return &gui.Link{Label: translate(label), URL: target}
	}

	return fallback
}

// OriginOf reads an origin: the key of the label of its screen and its
// address, cleaned. The address is honoured only when it is relative to the
// web interface (no scheme, no host), bounded, and names a screen that may
// be an origin; anything else is no origin.
func OriginOf(from string) (label, target string, ok bool) {
	if from == "" || len(from) > MaxOriginLength || !strings.HasPrefix(from, constants.WebuiPrefix+"/") {
		return "", "", false
	}

	u, err := url.Parse(from)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" {
		return "", "", false
	}

	u.Fragment = ""

	var match mux.RouteMatch

	req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: strings.TrimPrefix(u.Path, constants.WebuiPrefix)}}
	if !originRouter().Match(req, &match) || match.MatchErr != nil || match.Route == nil {
		return "", "", false
	}

	return match.Route.GetName(), u.String(), true
}
