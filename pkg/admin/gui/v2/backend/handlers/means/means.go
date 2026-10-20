// Package means serves the authentication means of the owner of a sheet: the
// "Authentication" tab, the page of a means, the form that adds or edits one
// and the confirmation of a deletion. The owner is a local server (the
// identity it presents to the partners) or one of its local accounts (what
// the partner presents), a remote partner (what the gateway checks of it) or
// one of its remote accounts (what the gateway presents to it): the screen of
// the owner describes it (Owner), the pages are the same.
//
// The pages never show a secret: a certificate is summarised by its subject,
// its validity and its fingerprint, a key by its kind and the fingerprint of
// its public part, a password by "set". On edition, an empty secret keeps the
// current one.
package means

import (
	"net/http"
	"slices"
	"strconv"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

const (
	// Stylesheet of the pages of a sheet (tabs, intro, pasted content, PEM
	// block, banners and action pages), under the web interface; the screen
	// of the owner links it before its own.
	Stylesheet = "/static_v2/sheet/sheet.css"

	// NewSuffix, EditSuffix and DeleteSuffix are the suffixes of the routes of
	// the pages of a means, under the "Authentication" tab of the owner.
	NewSuffix    = "/new"
	EditSuffix   = "/edit"
	DeleteSuffix = "/delete"

	// KindCred is the kind the redirection after an action carries
	// (liststate.KindParam), so that the owner's screen renders the right
	// notification.
	KindCred = "cred"

	// Prefix of the fields of the form of a means inside the form of an
	// account ("means_type").
	Prefix = "means_"

	sortName = "name"
	sortType = "type"

	typeKeyPrefix = "auth.type."
)

//nolint:gochecknoglobals // reference data of the pages
var (
	sortKeys = []string{sortName, sortType}

	counter = liststate.Counter{
		One: "auth.count_one", Many: "auth.count", ShownOne: "auth.count_shown_one", ShownMany: "auth.count_shown",
	}
)

// Rights of the current user on the owner: reading is implied by reaching
// the pages; writing and deleting are checked again by the router on each
// action. The pages hide what the rights forbid.
type Rights struct{ Write, Delete bool }

// Tab is an entry of the tab strip of a sheet: a link to a page, with the
// number of objects it lists when it is a list.
type Tab struct {
	Label    string
	URL      string
	Count    uint64
	HasCount bool
	Current  bool
	// Disabled is a tab of a sheet whose object does not exist yet: shown
	// greyed, with Hint saying why (Driss, 2026-10-20: the tab strip stays
	// visible on every page of a sheet, the form of a new object included).
	Disabled bool
	Hint     string
}

// PendingTabs is the tab strip of the form of an object not created yet:
// its details, current, then the tabs it will have once saved, greyed.
func PendingTabs(translate gui.Translator, details string, later ...string) []Tab {
	tabs := []Tab{{Label: details, Current: true}}

	for _, label := range later {
		tabs = append(tabs, Tab{Label: label, Disabled: true, Hint: translate("auth.tab_after_creation")})
	}

	return tabs
}

// Texts are the catalogue keys of the sentences that name the owner on the
// pages: the intro of the tab, the empty state, the consequence of a
// deletion, the state of a protocol that asks no means from the owner, the
// label of the way out of the tab and the label of the tab strip.
type Texts struct {
	Intro     string
	Empty     string
	Delete    string
	NoneTitle string
	NoneText  string
	Back      string
	Tabs      string
}

// Chrome is the screen of the owner around the pages: its section and its
// entry of the sidebar, its documentation page and its stylesheets.
type Chrome struct {
	Section     string
	Menu        string
	Doc         string
	Stylesheets []string
}

// Owner is the holder of the means the pages concern, as the screen of the
// owner describes it: what the model needs to read and write the means, what
// the protocol admits, the addresses of the pages, the sentences that name
// the owner, its tab strip, its breadcrumb and its chrome. The screen builds
// one per request.
type Owner struct {
	// Table is the owner as the model sees it: it reads its means, marks a
	// new one as its own and conditions a query.
	Table    model.CredOwnerTable
	Protocol string
	// Verified says which registry of the model holds the handlers of the
	// means: the internal one when the means are what the gateway checks of
	// the owner (a local account, a remote partner), the external one when
	// they are what the owner presents (a local server, a remote account).
	Verified bool
	// Allowed are the kinds of means the protocol admits for the owner.
	Allowed []string
	// AuthURL is the "Authentication" tab of the owner, without the list
	// state; BackURL is the way out of the tab, with it.
	AuthURL string
	BackURL string
	// Carried is the state of the list of the screen (?ret=), carried by
	// every link and form of the pages.
	Carried string
	// Title is the title of the sheet of the owner; Label names the owner
	// in a sentence ("the server « x »"). Both are translated.
	Title string
	Label string
	Texts Texts
	// Rights of the current user on the owner.
	Rights Rights
	// Tabs is the tab strip of the sheet of the owner, with the current tab.
	Tabs func(current string) ([]Tab, error)
	// Crumbs is the breadcrumb of a page of the sheet of the owner, with the
	// action page when there is one.
	Crumbs func(last string) []gui.Crumb
	Chrome Chrome
}

// WithRet carries the list state on an address of the pages.
func (o *Owner) WithRet(u string) string { return liststate.WithRet(u, o.Carried) }

// CredURL is a page of one means of the owner.
func (o *Owner) CredURL(id int64, suffix string) string {
	return o.AuthURL + "/" + strconv.FormatInt(id, 10) + suffix
}

// Unique tells whether the owner may hold a single means of the kind, from
// the model's own handler of the kind.
func (o *Owner) Unique(typ string) bool {
	var handler authentication.Handler
	if o.Verified {
		handler = authentication.GetInternalAuthHandler(typ, o.Protocol)
	} else {
		handler = authentication.GetExternalAuthMethod(typ, o.Protocol)
	}

	return handler != nil && handler.CanOnlyHaveOne()
}

// Remaining are the kinds one can still add to the owner: a unique kind
// already present is left out.
func (o *Owner) Remaining(creds []*model.Credential) []string {
	var out []string

	for _, typ := range o.Allowed {
		present := slices.ContainsFunc(creds, func(c *model.Credential) bool { return c.Type == typ })
		if !present || !o.Unique(typ) {
			out = append(out, typ)
		}
	}

	return out
}

// screen is a page of the sheet of the owner: its chrome around the data,
// with the result of the previous action when the page is a tab.
func (o *Owner) screen(title, last string, status int, notices []gui.Notice) common.Screen {
	return common.Screen{
		Title:   title,
		Section: o.Chrome.Section,
		Menu:    o.Chrome.Menu,
		Doc:     o.Chrome.Doc,
		Crumbs:  o.Crumbs(last),
		Status:  status,
		Notices: notices,

		Stylesheets: o.Chrome.Stylesheets,
	}
}

// ---------------------------------------------------------------- kinds ----

// ServerTypes are the means a local server of the protocol presents to the
// partners (its identity): the table of the historic page, without the
// PeSIT pre-connection that the model refuses on a server. The legacy R66
// certificate is offered when the instance allows it.
func ServerTypes(proto string) []string {
	switch proto {
	case r66.R66:
		return []string{auth.Password}
	case r66.R66TLS:
		return withLegacy([]string{auth.Password, auth.TLSCertificate})
	case httpconst.HTTPS, ftp.FTPS, webdav.WebdavTLS, as2.AS2TLS, pesit.PesitTLS:
		return []string{auth.TLSCertificate}
	case sftp.SFTP:
		return []string{sftp.AuthSSHPrivateKey}
	default:
		return nil
	}
}

// LocalAccountTypes are the means a partner may present on a local account
// of a server of the protocol: a password always, a trusted certificate over
// TLS, a public key over SSH, the legacy R66 certificate when the instance
// allows it.
func LocalAccountTypes(proto string) []string {
	types := []string{auth.Password}

	switch proto {
	case r66.R66TLS:
		types = withLegacy(append(types, auth.TLSTrustedCertificate))
	case httpconst.HTTPS, ftp.FTPS, webdav.WebdavTLS, as2.AS2TLS, pesit.PesitTLS:
		types = append(types, auth.TLSTrustedCertificate)
	case sftp.SFTP:
		types = append(types, sftp.AuthSSHPublicKey)
	}

	return types
}

// PartnerTypes are the means the gateway checks of a remote partner of the
// protocol: the table of the historic page, plus the password of a PeSIT
// partner (the client checks the password the server presents when the
// partner holds one). The legacy R66 certificate is offered when the
// instance allows it.
func PartnerTypes(proto string) []string {
	switch proto {
	case r66.R66, pesit.Pesit:
		return []string{auth.Password}
	case r66.R66TLS:
		return withLegacy([]string{auth.Password, auth.TLSTrustedCertificate})
	case pesit.PesitTLS:
		return []string{auth.Password, auth.TLSTrustedCertificate}
	case httpconst.HTTPS, ftp.FTPS, webdav.WebdavTLS, as2.AS2TLS:
		return []string{auth.TLSTrustedCertificate}
	case sftp.SFTP:
		return []string{sftp.AuthSSHPublicKey}
	default:
		return nil
	}
}

// RemoteAccountTypes are the means the gateway presents to a remote partner
// of the protocol on one of its accounts: a password always (the PeSIT
// client sends it whatever the encryption), a certificate and its key over
// TLS, a private key over SSH, the PeSIT pre-connection (a login and a
// password the partner may expect before the connection), the legacy R66
// certificate when the instance allows it.
func RemoteAccountTypes(proto string) []string {
	types := []string{auth.Password}

	switch proto {
	case r66.R66TLS:
		types = withLegacy(append(types, auth.TLSCertificate))
	case httpconst.HTTPS, ftp.FTPS, webdav.WebdavTLS, as2.AS2TLS:
		types = append(types, auth.TLSCertificate)
	case pesit.Pesit:
		types = append(types, pesit.PreConnectionAuth)
	case pesit.PesitTLS:
		types = append(types, auth.TLSCertificate, pesit.PreConnectionAuth)
	case sftp.SFTP:
		types = append(types, sftp.AuthSSHPrivateKey)
	}

	return types
}

func withLegacy(types []string) []string {
	if compatibility.IsLegacyR66CertificateAllowed {
		return append(types, r66auth.AuthLegacyCertificate)
	}

	return types
}

// TypeLabel names a kind of means; a kind the pages do not know keeps its key.
func TypeLabel(translate gui.Translator, typ string) string {
	if _, ok := families[typ]; ok {
		return translate(typeKeyPrefix + typ)
	}

	return typ
}

// ---------------------------------------------------------------- pages ----

// Pages serves the pages of the means of an owner. The screen of the owner
// resolves the owner from the route, then hands it over with the request.
type Pages struct {
	DB     *database.DB
	Logger *log.Logger
	Shell  *common.Shell
}

// credential reads the {cid} of the route among the means of the owner; a
// missing one renders the localised "not found" page and returns false.
func (p *Pages) credential(w http.ResponseWriter, r *http.Request, own *Owner) (*model.Credential, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["cid"], 10, 64)

	var cred model.Credential

	err := p.DB.Get(&cred, "id=?", id).And(own.Table.GetCredCond()).Run()
	if parseErr == nil && err == nil {
		return &cred, true
	}

	if err != nil && !database.IsNotFound(err) {
		p.Logger.Errorf("failed to retrieve the authentication means %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	}

	translate := p.Shell.Translator(r)
	title := translate("auth.not_found.title")

	p.NotFound(w, r, own.Chrome, own.Crumbs(title), NotFoundData{
		Title:     title,
		Text:      translate("auth.not_found.text"),
		Icon:      "key",
		BackLabel: translate("action.back"),
		BackURL:   own.WithRet(own.AuthURL),
	})

	return nil, false
}

// NotFoundData is a localised "not found" page: what is missing and the way
// back.
type NotFoundData struct {
	Title     string
	Text      string
	Icon      string
	BackLabel string
	BackURL   string
}

// NotFound renders the localised "not found" page of a screen (404), in its
// chrome, with the breadcrumb of the missing object.
//
//nolint:gocritic // hugeParam: one chrome per request
func (p *Pages) NotFound(w http.ResponseWriter, r *http.Request, chrome Chrome, crumbs []gui.Crumb, data NotFoundData) {
	p.Shell.Render(w, p.Logger, r, common.Screen{
		Title:   data.Title,
		Section: chrome.Section,
		Menu:    chrome.Menu,
		Doc:     chrome.Doc,
		Crumbs:  crumbs,
		Status:  http.StatusNotFound,
	}, NotFoundView(data))
}

// tabs is the tab strip of the owner on its "Authentication" tab, or an
// internal error.
func (p *Pages) tabs(w http.ResponseWriter, own *Owner) ([]Tab, bool) {
	tabs, err := own.Tabs(tabAuth)
	if err != nil {
		p.Logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	}

	return tabs, true
}

// tabAuth is the current tab of every page of the means.
const tabAuth = "auth"
