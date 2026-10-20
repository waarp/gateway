//go:build e2e

package admin_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// Timings of the smoke test. A screen counts as settled once no request has
// been in flight for e2eQuiet; the state refresh of the service status
// screen is awaited explicitly, its period being longer than that.
const (
	e2eQuiet         = time.Second
	e2eSettleTimeout = 30 * time.Second
	e2ePollInterval  = 50 * time.Millisecond
	e2eTimeoutMs     = 60_000.0
	e2eRefreshMs     = 15_000.0

	e2eUsername = "smoke"
	e2ePassword = "smoke_password"
)

// e2eScreen is a screen of the reworked web interface, with the number of
// tables and of tile tabs it must render and the query of the fragment it
// refreshes on its own (empty when it has none). The smoke user holds the
// rights on the servers, which the local servers screens require.
type e2eScreen struct {
	path     string
	tables   int
	tabs     int
	fragment string
	outside  bool // the screen has no entry in the sidebar (the home page)
}

func e2eScreens() []e2eScreen {
	return []e2eScreen{
		{path: "/webui/home", outside: true},
		{path: "/webui/about", outside: true},
		{path: "/webui/account", tables: 1, outside: true},
		{path: "/webui/services", tables: 3, tabs: 3, fragment: "fragment=state"},
		{path: "/webui/servers", tables: 1},
		{path: "/webui/servers/1"},
		{path: "/webui/servers/new"},
		{path: "/webui/servers/1/edit"},
		{path: "/webui/servers/1/delete"},
		// The sheet of the SFTP server: its host key, its accounts, the
		// sheet of the partner account and its means, the pages that add
		// a means and an account; the certificate of the HTTPS server.
		{path: "/webui/servers/1/auth", tables: 1},
		{path: "/webui/servers/1/auth/1"},
		{path: "/webui/servers/1/auth/new"},
		{path: "/webui/servers/1/accounts", tables: 1},
		{path: "/webui/servers/1/accounts/new"},
		{path: "/webui/servers/1/accounts/1"},
		{path: "/webui/servers/1/accounts/1/edit"},
		{path: "/webui/servers/1/accounts/1/delete"},
		{path: "/webui/servers/1/accounts/1/auth", tables: 1},
		{path: "/webui/servers/1/accounts/1/auth/2"},
		{path: "/webui/servers/1/accounts/1/auth/3/edit"},
		{path: "/webui/servers/1/accounts/1/auth/2/delete"},
		{path: "/webui/servers/1/accounts/1/auth/new"},
		{path: "/webui/servers/3/auth", tables: 1},
		{path: "/webui/servers/3/auth/4"},
		// The local clients: the list, the details of the SFTP client (the
		// SFTP partner it reaches: a table), the pages that add, edit and
		// delete one.
		{path: "/webui/clients", tables: 1},
		{path: "/webui/clients/1", tables: 1},
		{path: "/webui/clients/new", tables: 1}, // the table of the retries
		{path: "/webui/clients/1/edit", tables: 1},
		{path: "/webui/clients/1/delete"},
		// The remote partners (#714): the list, the sheet of the SFTP
		// partner (its details, the host key the gateway checks, the pages
		// that add a means), the sheet of the R66 over TLS partner (no
		// means yet), the pages that add, edit and delete a partner.
		{path: "/webui/partners", tables: 1},
		{path: "/webui/partners/1"},
		{path: "/webui/partners/new"},
		{path: "/webui/partners/1/edit"},
		{path: "/webui/partners/1/delete"},
		{path: "/webui/partners/1/auth", tables: 1},
		{path: "/webui/partners/1/auth/5"},
		{path: "/webui/partners/1/auth/5/edit"},
		{path: "/webui/partners/1/auth/5/delete"},
		{path: "/webui/partners/1/auth/new"},
		{path: "/webui/partners/2"},
		{path: "/webui/partners/2/auth"},
		// The remote account of the SFTP partner: the accounts tab, the
		// sheet of the account and its password (means 6), the pages that
		// add, edit and delete an account and a means.
		{path: "/webui/partners/1/accounts", tables: 1},
		{path: "/webui/partners/1/accounts/new"},
		{path: "/webui/partners/1/accounts/1"},
		{path: "/webui/partners/1/accounts/1/edit"},
		{path: "/webui/partners/1/accounts/1/delete"},
		{path: "/webui/partners/1/accounts/1/auth", tables: 1},
		{path: "/webui/partners/1/accounts/1/auth/6"},
		{path: "/webui/partners/1/accounts/1/auth/6/edit"},
		{path: "/webui/partners/1/accounts/1/auth/6/delete"},
		{path: "/webui/partners/1/accounts/1/auth/new"},
	}
}

// e2eService is a stub registered in the service lists of the gateway, so
// that the screens show rows without opening any port.
type e2eService struct {
	name   string
	state  utils.StateCode
	reason string
}

func (s *e2eService) Name() string                     { return s.name }
func (s *e2eService) Start() error                     { return nil }
func (s *e2eService) Stop(context.Context) error       { return nil }
func (s *e2eService) State() (utils.StateCode, string) { return s.state, s.reason }

func (s *e2eService) InitTransfer(*pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return nil, pipeline.NewError(0, "stub client: no transfer")
}

// registerE2EServices fills the service lists with every state the service
// status screen renders, and empties them afterwards.
func registerE2EServices(t *testing.T) {
	t.Helper()

	services.Core.Add(&e2eService{name: "Database", state: utils.StateRunning})
	services.Core.Add(&e2eService{name: "Controller", state: utils.StateRunning})
	services.Servers.Add(&model.LocalAgent{Identifier: model.Identifier{ID: 1}},
		&e2eService{name: "srv-sftp-01", state: utils.StateRunning})
	services.Servers.Add(&model.LocalAgent{Identifier: model.Identifier{ID: 2}},
		&e2eService{name: "srv-r66-01", state: utils.StateError, reason: "listen tcp: address already in use"})
	services.Servers.Add(&model.LocalAgent{Identifier: model.Identifier{ID: 3}},
		&e2eService{name: "srv-https-01", state: utils.StateOffline})
	services.Clients.Add(&model.Client{Identifier: model.Identifier{ID: 1}},
		&e2eService{name: "cli-sftp", state: utils.StateRunning})

	t.Cleanup(func() {
		services.Core = nil
		services.Servers.Reset()
		services.Clients.Reset()
	})
}

// e2eTracker counts the requests in flight and records the failed ones by
// screen. Playwright calls its handlers on its own dispatch goroutine.
type e2eTracker struct {
	mu       sync.Mutex
	baseURL  string
	screen   string
	inFlight map[playwright.Request]struct{}
	last     time.Time
	failures map[string][]string
}

func trackRequests(page playwright.Page, baseURL string) *e2eTracker {
	tracker := &e2eTracker{
		mu:       sync.Mutex{},
		baseURL:  baseURL,
		screen:   "",
		inFlight: make(map[playwright.Request]struct{}),
		last:     time.Time{},
		failures: make(map[string][]string),
	}

	page.OnRequest(tracker.onRequest)
	page.OnResponse(tracker.onResponse)
	page.OnRequestFinished(tracker.onDone)
	page.OnRequestFailed(tracker.onFailed)

	return tracker
}

func (tr *e2eTracker) isOurs(req playwright.Request) bool {
	return strings.HasPrefix(req.URL(), tr.baseURL)
}

func (tr *e2eTracker) onRequest(req playwright.Request) {
	if !tr.isOurs(req) {
		return
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.inFlight[req] = struct{}{}
	tr.last = time.Now()
}

// onResponse records every response of the backend that is an error, pages
// and static files included.
func (tr *e2eTracker) onResponse(res playwright.Response) {
	if res.Status() < http.StatusBadRequest || !tr.isOurs(res.Request()) {
		return
	}

	tr.record(res.Request(), fmt.Sprintf("status %d", res.Status()))
}

func (tr *e2eTracker) onFailed(req playwright.Request) {
	if !tr.isOurs(req) {
		return
	}

	tr.record(req, fmt.Sprintf("failed: %v", req.Failure()))
	tr.onDone(req)
}

func (tr *e2eTracker) onDone(req playwright.Request) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, ok := tr.inFlight[req]; ok {
		delete(tr.inFlight, req)
		tr.last = time.Now()
	}
}

// record keeps a failure under the current screen: settle runs before every
// screen change, so no request outlives its screen.
func (tr *e2eTracker) record(req playwright.Request, reason string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.failures[tr.screen] = append(tr.failures[tr.screen],
		fmt.Sprintf("%s %s: %s", req.Method(), req.URL(), reason))
}

// enter makes screen the current screen.
func (tr *e2eTracker) enter(screen string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.screen = screen
}

func (tr *e2eTracker) settled() (bool, []string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	pending := make([]string, 0, len(tr.inFlight))
	for req := range tr.inFlight {
		pending = append(pending, req.URL())
	}

	return len(pending) == 0 && time.Since(tr.last) >= e2eQuiet, pending
}

// settle waits until no request has been in flight for e2eQuiet, counted
// from the call at the earliest.
func (tr *e2eTracker) settle() error {
	start := time.Now()

	tr.mu.Lock()
	tr.last = start
	tr.mu.Unlock()

	deadline := start.Add(e2eSettleTimeout)

	for {
		done, pending := tr.settled()
		if done {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("requests still in flight: %v", pending) //nolint:err113 // test message
		}

		time.Sleep(e2ePollInterval)
	}
}

// failuresOf returns the failures recorded for screen.
func (tr *e2eTracker) failuresOf(screen string) []string {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	return slices.Clone(tr.failures[screen])
}

// startE2EServer serves the administration interface (REST API and web
// interface) on a fresh database holding one user with the rights on the
// servers and the three servers whose services registerE2EServices stubs.
func startE2EServer(t *testing.T) string {
	t.Helper()

	db := dbtest.TestDatabase(t)

	hash, err := bcrypt.GenerateFromPassword([]byte(e2ePassword), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{
		Username: e2eUsername, PasswordHash: string(hash),
		Permissions: model.PermServersRead | model.PermServersWrite | model.PermServersDelete |
			model.PermPartnersRead | model.PermPartnersWrite | model.PermPartnersDelete,
	}).Run())

	servers := []*model.LocalAgent{
		{Name: "srv-sftp-01", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)},
		{Name: "srv-r66-01", Protocol: "r66", Address: types.Addr("127.0.0.1", 6666)},
		{Name: "srv-https-01", Protocol: "https", Address: types.Addr("127.0.0.1", 8443)},
	}

	for i, server := range servers {
		require.NoError(t, db.Insert(server).Run())
		require.EqualValues(t, i+1, server.ID, "the stub services are registered by identifier")
	}

	// The SFTP client whose service registerE2EServices stubs.
	client := &model.Client{Name: "cli-sftp", Protocol: "sftp", NbOfAttempts: 3, FirstRetryDelay: 30, RetryIncrementFactor: 2}
	require.NoError(t, db.Insert(client).Run())
	require.EqualValues(t, 1, client.ID, "the stub service is registered by identifier")

	// The sheet of the SFTP server: its host key (means 1), an account with
	// a public key and a password (means 2 and 3) and an account without
	// any means; the certificate of the HTTPS server (means 4). The screens
	// are visited by identifier.
	partner := &model.LocalAccount{LocalAgentID: servers[0].ID, Login: "partner", IPAddresses: types.IPList{"10.20.30.4"}}
	require.NoError(t, db.Insert(partner).Run())
	require.NoError(t, db.Insert(&model.LocalAccount{LocalAgentID: servers[0].ID, Login: "nobody"}).Run())

	for i, cred := range []*model.Credential{
		{LocalAgentID: servers[0].NullableID(), Name: "cle-hote", Type: "ssh_private_key", Value: testhelpers.RSAPk},
		{LocalAccountID: partner.NullableID(), Name: "cle-partner", Type: "ssh_public_key", Value: testhelpers.SSHPbk},
		{LocalAccountID: partner.NullableID(), Name: "mot-de-passe", Type: "password", Value: e2ePassword},
		{LocalAgentID: servers[2].NullableID(), Name: "cert-localhost", Type: "tls_certificate",
			Value: testhelpers.LocalhostCert, Value2: testhelpers.LocalhostKey},
	} {
		require.NoError(t, db.Insert(cred).Run())
		require.EqualValues(t, i+1, cred.ID, "the screens are visited by identifier")
	}

	// The remote partners (#714): an SFTP partner with the host key the
	// gateway checks (means 5) and a remote account with its password (means
	// 6), an R66 over TLS partner without anything yet. The SFTP client then
	// reaches a partner: its sheet shows the table.
	partners := []*model.RemoteAgent{
		{Name: "part-sftp-01", Protocol: "sftp", Address: types.Addr("127.0.0.1", 2222), ProtoConfig: map[string]any{"useStat": true}},
		{Name: "part-r66-tls", Protocol: "r66-tls", Address: types.Addr("127.0.0.2", 6667), ProtoConfig: map[string]any{"serverLogin": "waarp"}},
	}

	for i, partner := range partners {
		require.NoError(t, db.Insert(partner).Run())
		require.EqualValues(t, i+1, partner.ID, "the screens are visited by identifier")
	}

	remote := &model.RemoteAccount{RemoteAgentID: partners[0].ID, Login: "gateway-prod"}
	require.NoError(t, db.Insert(remote).Run())
	require.EqualValues(t, 1, remote.ID)

	for i, cred := range []*model.Credential{
		{RemoteAgentID: partners[0].NullableID(), Name: "cle-hote-partenaire", Type: "ssh_public_key", Value: testhelpers.SSHPbk},
		{RemoteAccountID: remote.NullableID(), Name: "mot-de-passe", Type: "password", Value: e2ePassword},
	} {
		require.NoError(t, db.Insert(cred).Run())
		require.EqualValues(t, i+5, cred.ID, "the screens are visited by identifier")
	}

	registerE2EServices(t)

	server := httptest.NewServer(admin.MakeHandler(testhelpers.GetTestLogger(t), db))
	t.Cleanup(server.Close)

	return server.URL
}

//nolint:ireturn // playwright.Page is the library's own interface type
func newE2EPage(t *testing.T) playwright.Page {
	t.Helper()

	driver, err := playwright.Run()
	require.NoError(t, err, "cannot start Playwright (see ./make.sh test e2e)")
	t.Cleanup(func() { assert.NoError(t, driver.Stop()) })

	browser, err := driver.Chromium.Launch()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, browser.Close()) })

	page, err := browser.NewPage()
	require.NoError(t, err)

	page.SetDefaultTimeout(e2eTimeoutMs)
	page.SetDefaultNavigationTimeout(e2eTimeoutMs)

	return page
}

// expectURL waits for the page to land on exactly want.
func expectURL(t *testing.T, page playwright.Page, want string) {
	t.Helper()

	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(want) + "$")
	err := playwright.NewPlaywrightAssertions(e2eTimeoutMs).Page(page).ToHaveURL(pattern)
	require.NoError(t, err)
}

// e2eSession is a browser page on the web interface served at baseURL.
type e2eSession struct {
	page    playwright.Page
	tracker *e2eTracker
	baseURL string
}

// visit opens path as the current screen and waits for it to settle.
func (s *e2eSession) visit(t *testing.T, path string) {
	t.Helper()

	s.tracker.enter(path)

	_, err := s.page.Goto(s.baseURL + path)
	require.NoError(t, err)
	require.NoError(t, s.tracker.settle())
}

func (s *e2eSession) login(t *testing.T) {
	t.Helper()

	s.visit(t, "/webui/login")

	require.NoError(t, s.page.Locator("#username").Fill(e2eUsername))
	require.NoError(t, s.page.Locator("#password").Fill(e2ePassword))
	require.NoError(t, s.page.Locator("form button[type=submit]").Click())

	expectURL(t, s.page, s.baseURL+"/webui/home")
	require.NoError(t, s.tracker.settle())

	// The sign-in page is a historic one, checked once reworked: its failed
	// requests are reported, not judged. The home page it lands on is a
	// reworked screen, checked like every other one.
	if failures := s.tracker.failuresOf("/webui/login"); len(failures) > 0 {
		t.Logf("failed requests of the historic sign-in page: %v", failures)
	}
}

// count returns the number of elements matching selector.
func (s *e2eSession) count(t *testing.T, selector string) int {
	t.Helper()

	n, err := s.page.Locator(selector).Count()
	require.NoError(t, err)

	return n
}

// attribute returns the value of an attribute of the element.
func attribute(t *testing.T, element playwright.Locator, name string) string {
	t.Helper()

	value, err := element.GetAttribute(name)
	require.NoError(t, err)

	return value
}

func (s *e2eSession) check(t *testing.T, screen e2eScreen) {
	t.Helper()

	s.visit(t, screen.path)
	expectURL(t, s.page, s.baseURL+screen.path)

	// The kit chrome and the screen.
	assert.Equal(t, 1, s.count(t, "a.w-skip"), "skip link")
	assert.Equal(t, 1, s.count(t, "main#contenu"), "main landmark")
	assert.Equal(t, 1, s.count(t, "h1.w-page-header__title"), "page title")
	entries := 1
	if screen.outside {
		entries = 0
	}

	assert.Equal(t, entries, s.count(t, `nav:not(.w-tabs) a[aria-current="page"]`), "active sidebar entry")
	assert.Equal(t, screen.tables, s.count(t, "table.w-table"), "tables")

	// A sheet shows its tab strip, one tab being the page: a link, or a
	// plain span on the form of an object not created yet.
	if s.count(t, "nav.w-tabs") > 0 {
		assert.Equal(t, 1, s.count(t, `nav.w-tabs [aria-current="page"]`), "current tab")
	}

	// The first Tab reaches the skip link.
	require.NoError(t, s.page.Keyboard().Press("Tab"))
	focused, err := s.page.Evaluate(`document.activeElement.classList.contains("w-skip")`)
	require.NoError(t, err)
	assert.Equal(t, true, focused, "the first Tab must reach the skip link")

	// The tiles are tabs: the last one shows its panel alone, and that choice
	// survives a refresh of the fragment.
	if screen.tabs > 0 {
		tabs := s.page.Locator(`[role="tablist"] [role="tab"]`)
		require.Equal(t, screen.tabs, s.count(t, `[role="tablist"] [role="tab"]`), "tile tabs")
		require.NoError(t, tabs.Last().Click())
		require.NoError(t, s.tracker.settle())
		assert.Equal(t, 1, s.count(t, `[role="tabpanel"]:visible`), "one panel shown at a time")
		assert.Equal(t, "true", attribute(t, tabs.Last(), "aria-selected"))
	}

	// One refresh of the screen's own fragment, served without error, that
	// keeps the group shown.
	if screen.fragment != "" {
		response, err := s.page.ExpectResponse(func(res playwright.Response) bool {
			return strings.Contains(res.URL(), screen.fragment)
		}, func() error { return nil }, playwright.PageExpectResponseOptions{Timeout: playwright.Float(e2eRefreshMs)})
		require.NoError(t, err, "the screen must refresh its fragment on its own")
		assert.Equal(t, http.StatusOK, response.Status())
		require.NoError(t, s.tracker.settle())
		assert.Equal(t, 0, s.count(t, "#etat .w-banner"), "no error message after a successful refresh")

		if screen.tabs > 0 {
			assert.Equal(t, "true", attribute(t, s.page.Locator(`[role="tablist"] [role="tab"]`).Last(), "aria-selected"),
				"the group shown must survive the refresh")
			assert.Equal(t, 1, s.count(t, `[role="tabpanel"]:visible`))
		}
	}

	// A sortable column header, sorted ascending by default, sorts descending
	// once clicked (through the fragment or a reload, served without error).
	if headers := s.count(t, `th[aria-sort="ascending"] a:visible`); headers > 0 {
		require.NoError(t, s.page.Locator(`th[aria-sort="ascending"] a:visible`).First().Click())
		require.NoError(t, s.tracker.settle())
		assert.Equal(t, 1, s.count(t, `th[aria-sort="descending"]:visible`), "sorted descending")
	}

	// The filters of a list apply as soon as one changes; the "⋯" menu of a
	// row opens and closes from the keyboard.
	if s.count(t, `form[data-autosubmit] details.w-disclose`) > 0 {
		require.NoError(t, s.page.Locator(`form[data-autosubmit] details.w-disclose > summary`).Click())
		require.NoError(t, s.page.Locator(`form[data-autosubmit] input[name="protocol"]`).First().Check())
		require.NoError(t, s.tracker.settle())
		assert.Equal(t, 1, s.count(t, `form[data-autosubmit] input[name="protocol"]:checked`), "the filter is kept after the reload")
		assert.Equal(t, 1, s.count(t, `form[data-autosubmit] details.w-disclose[open]`), "the filters stay open while one is active")
	}

	if s.count(t, `button[popovertarget]`) > 0 {
		menu := s.page.Locator(`button[popovertarget]`).First()
		require.NoError(t, menu.Focus())
		require.NoError(t, s.page.Keyboard().Press("Enter"))
		assert.Equal(t, 1, s.count(t, `.w-actions:popover-open`), "the menu opens from the keyboard")
		require.NoError(t, s.page.Keyboard().Press("Escape"))
		assert.Equal(t, 0, s.count(t, `.w-actions:popover-open`), "and closes with Escape")
	}

	// An ordered list is a Tags field of the kit: with the SFTP family shown,
	// typing in the field of the key exchanges filters the suggestions and
	// Enter adds the match, which the hidden select then sends first.
	if s.count(t, `#sftp_kex`) > 0 {
		_, err := s.page.Locator(`select[data-switch]`).First().SelectOption(playwright.SelectOptionValues{Values: playwright.StringSlice("sftp")})
		require.NoError(t, err)
		// The card of the options folds when nothing is set: open it, as a user would.
		_, err = s.page.Evaluate(`document.querySelector('details[data-case="sftp"]').open = true`)
		require.NoError(t, err)
		require.NoError(t, s.page.Locator(`#sftp_kex ~ .w-tags input`).Focus())
		require.NoError(t, s.page.Keyboard().Type("curve"))
		assert.Equal(t, 1, s.count(t, `#sftp_kex-menu [role="option"]`), "typing filters the suggestions")
		require.NoError(t, s.page.Keyboard().Press("Enter"))
		chosen, err := s.page.Evaluate(`[...document.querySelectorAll("#sftp_kex option:checked")].map(o => o.value)`)
		require.NoError(t, err)
		assert.Equal(t, []any{"curve25519-sha256@libssh.org"}, chosen, "Enter adds the match to the hidden select")
	}

	// The retries of a client are drawn live: the sentence and the marks of
	// the timeline follow the three fields, and the script computes the
	// waits exactly as the model does.
	if s.count(t, `[data-retry]`) > 0 {
		_, err := s.page.Evaluate(`document.querySelector('details.w-card--fold:has([data-retry])').open = true`)
		require.NoError(t, err)
		require.NoError(t, s.page.Locator(`#attempts`).Fill("5"))
		require.NoError(t, s.page.Locator(`#delay_s`).Fill("30"))
		require.NoError(t, s.page.Locator(`#factor`).Fill("2"))
		assert.Equal(t, 5, s.count(t, `.w-retry__mark`), "one mark per attempt")
		assert.Equal(t, 5, s.count(t, `.w-retry__rows tr`), "one row per attempt")
		sentence, err := s.page.Locator(`.w-retry__text`).TextContent()
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(sentence, "5 "), "the sentence counts the attempts: %q", sentence)

		for _, c := range []struct {
			attempts int8
			delay    int32
			factor   float32
		}{{5, 30, 2}, {3, 0, 1}, {12, 30, 1.5}, {127, 30, 2}, {5, 10, 0.5}, {1, 7, 3}} {
			// The browser returns whole numbers as int and the others as
			// float64: everything is compared as int64.
			got, err := s.page.Evaluate(fmt.Sprintf("window.wRetryWaits(%d, %d, %v)", c.attempts, c.delay, c.factor))
			require.NoError(t, err)

			values, ok := got.([]any)
			require.True(t, ok, "the script returns the waits as a list (%+v): %v", c, got)

			gotWaits := make([]int64, 0, len(values))
			for _, v := range values {
				switch n := v.(type) {
				case int:
					gotWaits = append(gotWaits, int64(n))
				case float64:
					gotWaits = append(gotWaits, int64(n))
				default:
					t.Fatalf("unexpected wait %v (%T)", v, v)
				}
			}

			want := []int64{}
			for _, w := range model.RetryWaits(c.attempts, c.delay, c.factor) {
				want = append(want, int64(w))
			}

			assert.Equal(t, want, gotWaits, "the script computes the waits as the model does (%+v)", c)
		}
	}

	assert.Empty(t, s.tracker.failuresOf(screen.path), "failed requests")

	t.Run("axe-core", func(t *testing.T) { s.checkAccessibility(t) })
}

// checkAccessibility runs the axe-core rules on the screen: no serious or
// critical violation. The rules come from the file named by AXE_CORE_JS,
// installed by helpers/playwright-bootstrap.sh.
func (s *e2eSession) checkAccessibility(t *testing.T) {
	t.Helper()

	path := os.Getenv("AXE_CORE_JS")
	if path == "" {
		t.Skip("AXE_CORE_JS is not set: run ./helpers/playwright-bootstrap.sh and source .playwright/env.sh")
	}

	_, err := s.page.AddScriptTag(playwright.PageAddScriptTagOptions{Path: playwright.String(path)})
	require.NoError(t, err)

	violations, err := s.page.Evaluate(`async () => {
		const result = await axe.run(document, { resultTypes: ["violations"] });
		return result.violations
			.filter((v) => v.impact === "serious" || v.impact === "critical")
			.map((v) => v.id + " (" + v.impact + "): " + v.help + " [" + v.nodes.map((n) => n.target.join(" ")).join(", ") + "]");
	}`)
	require.NoError(t, err)
	assert.Empty(t, violations, "serious or critical accessibility violations")
}

// TestE2ESmokeEveryScreen signs in as a user without any right and visits
// every screen of the reworked web interface against the real backend: any
// request answered with an error fails it.
//
//nolint:paralleltest // the test fills the global service lists of the gateway
func TestE2ESmokeEveryScreen(t *testing.T) {
	baseURL := startE2EServer(t)
	page := newE2EPage(t)
	session := &e2eSession{page: page, tracker: trackRequests(page, baseURL), baseURL: baseURL}

	session.login(t)

	for _, screen := range e2eScreens() {
		t.Run(screen.path, func(t *testing.T) { session.check(t, screen) })
	}
}
