package users

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermUsersRead
	permWrite = model.PermUsersRead | model.PermUsersWrite
	permAll   = model.PermUsersRead | model.PermUsersWrite | model.PermUsersDelete

	testPassword = "pa55word-of-tester"
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does. The signed-in user ("tester") is stored
// in the database, as the users screen acts on them too.
type suite struct {
	db     *database.DB
	router *mux.Router

	tester, viewer, ops, nobody *model.User
}

// hashOf hashes a password at the lowest cost: the fixtures do not measure
// bcrypt.
func hashOf(t *testing.T, password string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)

	return string(hash)
}

func newSuite(t *testing.T, perms model.PermsMask) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	s.tester = &model.User{Username: "tester", PasswordHash: hashOf(t, testPassword), Permissions: perms}
	require.NoError(t, s.db.Insert(s.tester).Run())

	// The default administrator of a fresh database is removed: the
	// scenarios decide who the administrators are.
	require.NoError(t, s.db.DeleteAll(&model.User{}).Where("username=?", "admin").Run())

	h := New(s.db, testhelpers.GetTestLogger(t))
	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/users").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/users/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/users/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/users/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/users/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/users/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/users/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/users/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures inserts three more users: a viewer who reads the servers, an
// operator who manages the partners and writes the rules, and a user
// without any right.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.viewer = &model.User{Username: "viewer", PasswordHash: hashOf(t, "viewer-pw"), Permissions: model.PermServersRead}
	s.ops = &model.User{
		Username: "ops", PasswordHash: hashOf(t, "ops-pw"),
		Permissions: model.PermPartnersRead | model.PermPartnersWrite | model.PermPartnersDelete |
			model.PermRulesRead | model.PermRulesWrite,
	}
	s.nobody = &model.User{Username: "nobody", PasswordHash: hashOf(t, "nobody-pw")}

	for _, user := range []*model.User{s.viewer, s.ops, s.nobody} {
		require.NoError(t, s.db.Insert(user).Run())
	}
}

// do sends a request as the tester, read again from the database (as the
// session middleware does), in the given language.
func (s *suite) do(t *testing.T, lang, method, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	var user model.User
	require.NoError(t, s.db.Get(&user, "id=?", s.tester.ID).Run())

	ctx := context.WithValue(context.Background(), constants.ContextUserKey, &user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	var req *http.Request
	if form != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequestWithContext(ctx, method, target, nil)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *suite) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()

	return s.do(t, "fr", http.MethodGet, target, nil)
}

func (s *suite) post(t *testing.T, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	if form == nil {
		form = url.Values{}
	}

	return s.do(t, "fr", http.MethodPost, target, form)
}

// user reads a user again from the database.
func (s *suite) user(t *testing.T, id int64) *model.User {
	t.Helper()

	user, err := internal.GetUserByID(s.db, id)
	require.NoError(t, err)

	return user
}

func (s *suite) count(t *testing.T) uint64 {
	t.Helper()

	n, err := s.db.Count(&model.User{}).Run()
	require.NoError(t, err)

	return n
}

// indexes asserts that the markers appear in the body in the given order,
// each one searched after the previous one.
func indexes(t *testing.T, body string, markers ...string) {
	t.Helper()

	from := 0

	for _, marker := range markers {
		pos := strings.Index(body[from:], marker)
		require.NotEqualf(t, -1, pos, "%q not found in order in the page", marker)
		from += pos + len(marker)
	}
}

// between is the part of the body between two markers.
func between(t *testing.T, body, from, to string) string {
	t.Helper()

	start := strings.Index(body, from)
	require.NotEqual(t, -1, start, from)
	end := strings.Index(body[start:], to)
	require.NotEqual(t, -1, end, to)

	return body[start : start+end]
}

func rows(body string) int { return strings.Count(body, "<tr>") - 1 }

func path(user *model.User, suffix string) string {
	return fmt.Sprintf("/webui/users/%d%s", user.ID, suffix)
}

// TestListFragment: the region of the list alone, for the live list of the kit (#766):
// the elements of the part "list", with the ids of the page, no shell; the
// page itself declares the live region and its filters form.
func TestListFragment(t *testing.T) {
	t.Parallel()

	s := newSuite(t, permAll)
	s.fixtures(t)

	rec := s.get(t, "/webui/users?fragment=list")
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.NotContains(t, body, "<html", "no shell around a fragment")
	assert.Contains(t, body, `<span class="w-toolbar__count" id="users-count" data-live-part="list" aria-live="polite">`)
	assert.Contains(t, body, `<div id="users-list" data-live-part="list" data-live-paged>`)

	body = s.get(t, "/webui/users").Body.String()
	assert.Contains(t, body, `<div data-live data-live-period="0">`, "read on the user's actions only")
	assert.Contains(t, body, `data-autosubmit data-live-filter>`, "the filters apply in place")
	assert.Contains(t, body, `<template data-live-error>`)
	assert.Contains(t, body, `live.js`)
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		rec := s.get(t, "/webui/users")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Utilisateurs · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/users" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.Contains(t, body, `<li><span aria-current="page">Utilisateurs</span></li>`)
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")

		// One coloured action, the search with its suggestions, the two filters, the count.
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/users/new">`)
		assert.Contains(t, body, `<option value="nobody"></option><option value="ops"></option><option value="tester"></option><option value="viewer"></option>`)
		assert.Contains(t, body, `<select class="w-select" id="domain" name="domain"><option value="" selected>Tous les domaines</option><option value="transfers">Transferts</option>`)
		assert.Contains(t, body, `<select class="w-select" id="right" name="right"><option value="" selected>Tous les droits</option><option value="read">Lecture</option>`)
		assert.Contains(t, body, `<option value="none">Aucun droit</option>`)
		assert.Contains(t, body, `aria-live="polite">4 utilisateurs</span>`)

		// Seven sortable headers, sorted by login ascending; the rows by login, each login a link.
		assert.Equal(t, 4, rows(body))
		assert.Contains(t, body, `<th scope="col" aria-sort="ascending"><a href="/webui/users?sort=name&amp;dir=desc">Utilisateur</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/users?sort=servers&amp;dir=asc">Serveurs</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/users?sort=admin&amp;dir=asc">Administration</a></th>`)
		indexes(t, body, "<tbody>", "nobody", "ops", "tester", "viewer", "</tbody>")
		assert.Contains(t, body, `<a class="w-cell-title w-nowrap" href="`+path(s.viewer, "")+`">viewer</a></td>`)

		// The signed-in user is marked; the rights in the notation of the product, with the legend.
		assert.Contains(t, body, `>tester</a> <span class="w-badge w-badge--neutral">Vous</span></td>`)
		assert.Equal(t, 1, strings.Count(body, "w-badge--neutral"), "one user is the signed-in one")
		assert.Contains(t, body, `>viewer</a></td><td><span class="w-mono">---</span></td><td><span class="w-mono">r--</span></td>`)
		assert.Contains(t, body, `>ops</a></td><td><span class="w-mono">---</span></td><td><span class="w-mono">---</span></td><td><span class="w-mono">rwd</span></td><td><span class="w-mono">rw-</span></td>`)
		assert.Contains(t, body, `>tester</a> <span class="w-badge w-badge--neutral">Vous</span></td><td><span class="w-mono">---</span></td>`)
		assert.Contains(t, body, `<td><span class="w-mono">rwd</span></td><td><span class="w-mono">---</span></td><td>`, "the tester manages the users only")
		assert.Contains(t, body, `<p class="users-legend w-muted">r : lecture · w : écriture · d : suppression</p>`)

		// The "⋯" menu: view, edit, delete last; never delete oneself.
		indexes(t, body, ">ops</a>", "Voir la fiche", "Modifier l&#39;utilisateur", "Supprimer l&#39;utilisateur", ">tester</a>")
		own := between(t, body, ">tester</a>", ">viewer</a>")
		assert.Contains(t, own, "Modifier l&#39;utilisateur")
		assert.NotContains(t, own, "Supprimer", "the signed-in user cannot delete their own account")
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permRead)
		s.fixtures(t)

		body := s.get(t, "/webui/users").Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action at all")
		assert.NotContains(t, body, "Modifier l&#39;utilisateur")
		assert.NotContains(t, body, "Supprimer l&#39;utilisateur")
		assert.Contains(t, body, "Voir la fiche")
	})

	t.Run("Given a search, filters and a sort", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		body := s.get(t, "/webui/users?q=VI").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, `aria-live="polite">4 utilisateurs · 1 affiché</span>`)

		body = s.get(t, "/webui/users?domain=partners&right=delete").Body.String()
		assert.Equal(t, 1, rows(body), "ops alone may delete partners")
		assert.Contains(t, body, `<details class="w-disclose" open>`)
		assert.Contains(t, body, `<option value="partners" selected>Partenaires distants</option>`)
		assert.Contains(t, body, `<option value="delete" selected>Suppression</option>`)
		assert.Contains(t, body, "Filtres · 2")
		assert.Contains(t, body, `<th scope="col" aria-sort="ascending"><a href="/webui/users?domain=partners&amp;right=delete&amp;sort=name&amp;dir=desc">`,
			"the sort links keep the filters")

		body = s.get(t, "/webui/users?right=none").Body.String()
		assert.Equal(t, 1, rows(body), "nobody holds no right at all")
		indexes(t, body, "<tbody>", "nobody", "</tbody>")

		body = s.get(t, "/webui/users?domain=servers&right=none").Body.String()
		assert.Equal(t, 3, rows(body), "nobody, ops and the tester have no right on the servers")

		body = s.get(t, "/webui/users?domain=rules").Body.String()
		assert.Equal(t, 1, rows(body), "ops alone has a right on the rules")

		body = s.get(t, "/webui/users?domain=transfers&right=delete").Body.String()
		assert.Contains(t, body, "Aucun utilisateur ne correspond aux filtres", "nobody may delete transfers")
		assert.Contains(t, body, `href="/webui/users?reset=1">Réinitialiser</a>`)

		body = s.get(t, "/webui/users?sort=servers&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "viewer", "nobody", "ops", "tester", "</tbody>")

		body = s.get(t, "/webui/users?sort=partners&dir=asc").Body.String()
		indexes(t, body, "<tbody>", "nobody", "tester", "viewer", "ops", "</tbody>")

		body = s.get(t, "/webui/users?q=zzz").Body.String()
		assert.Contains(t, body, "Aucun utilisateur ne correspond à « zzz »")
		assert.Contains(t, body, `href="/webui/users?reset=1">Effacer la recherche</a>`)
	})

	t.Run("Given the signed-in user alone", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permWrite)

		body := s.get(t, "/webui/users").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, `aria-live="polite">1 utilisateur</span>`)
		assert.Contains(t, body, `href="/webui/users/new">`)
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Given the sheet of the viewer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		rec := s.get(t, path(s.viewer, "")+"?ret=q%3Dvi")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Utilisateur « viewer » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="/webui/users?q=vi">Utilisateurs</a>`, "the breadcrumb keeps the list state")
		assert.Contains(t, body, `<dt>Identifiant</dt><dd><code class="w-code">viewer</code></dd>`)
		assert.NotContains(t, body, "Vous")
		assert.NotContains(t, body, "Dernier administrateur")
		assert.Contains(t, body, `<ul class="rights-list" aria-label="Droits de « viewer », par domaine : lecture, écriture, suppression">`)
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-key="users-rights">`, "the rights card folds, closed by default (#810)")
		assert.Regexp(t, `<span class="w-fold__meta">(Tous les droits|Aucun droit|1 droit|[0-9]+ droits)</span>`, body, "summed up in its heading")
		assert.Contains(t, body, `<strong class="rights-domain">Serveurs et clients locaux</strong> <span class="rights-granted">Lecture</span>`)
		assert.Contains(t, body, `<strong class="rights-domain">Transferts</strong> <span class="rights-granted w-muted">Aucun droit</span>`)
		assert.Contains(t, body, `href="/webui/static_v2/rights/rights.css"`, "the cards of the rights")
		assert.NotContains(t, body, "<table", "no table in a card")
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="`+path(s.viewer, "/edit")+`?ret=q%3Dvi">`)
		assert.Contains(t, body, `href="`+path(s.viewer, "/delete")+`?ret=q%3Dvi"`)
		assert.NotContains(t, body, "Voir la fiche", "no link to itself")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given the signed-in user's own sheet, the last administrator", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		body := s.get(t, path(s.tester, "")).Body.String()
		assert.Contains(t, body, `<code class="w-code">tester</code> <span class="w-badge w-badge--neutral">Vous</span>`)
		assert.Contains(t, body, "Dernier administrateur : le seul à détenir l&#39;écriture sur les utilisateurs.")
		assert.Contains(t, body, "Modifier l&#39;utilisateur")
		assert.NotContains(t, body, "Supprimer", "the signed-in user cannot delete their own account")
		assert.NotContains(t, body, "popovertarget", "no empty menu")
	})

	t.Run("Given a user with the read right only, in English", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permRead)
		s.fixtures(t)

		body := s.do(t, "en", http.MethodGet, path(s.ops, ""), nil).Body.String()
		assert.Contains(t, body, "<title>User “ops” · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<strong class="rights-domain">Remote partners</strong> <span class="rights-granted">Read, write and delete</span>`)
		assert.NotContains(t, body, "Edit the user")
		assert.NotContains(t, body, "popovertarget")
	})

	t.Run("Given an unknown user", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permRead)

		rec := s.get(t, "/webui/users/999?ret=page%3D2")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Cet utilisateur n&#39;existe pas ou a été supprimé.")
		assert.Contains(t, rec.Body.String(), `href="/webui/users?page=2"`, "the way back keeps the list state")
	})
}

func TestNewUser(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permWrite)

		rec := s.get(t, "/webui/users/new")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouvel utilisateur · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<form class="w-form" method="post" action="/webui/users/new" novalidate>`)
		assert.Contains(t, body, `id="login" name="login" value="" required autocomplete="username"`)
		assert.Contains(t, body, `id="password" name="password" type="password" required autocomplete="new-password"`)
		assert.Contains(t, body, `id="confirm" name="confirm" type="password" required autocomplete="new-password"`)
		assert.Contains(t, body, "Conservé haché ; personne ne pourra le relire. 72 caractères au plus.")
		assert.Equal(t, 17, strings.Count(body, `type="checkbox"`), "six domains, no delete right on the transfers")
		assert.Contains(t, body, `<fieldset class="rights-item"><legend class="rights-domain">Transferts</legend><div class="w-checks">`+
			`<label class="w-check"><input type="checkbox" id="transfers-r" name="transfers" value="r"> <span>Lecture</span></label>`)
		assert.NotContains(t, body, `name="transfers" value="d"`)
		assert.NotContains(t, body, "<table", "no table in a card")
		assert.Contains(t, body, "Les transferts ne se suppriment pas.")
		assert.NotContains(t, body, "checked", "nothing granted yet")
		assert.NotContains(t, body, "C&#39;est votre compte")
		assert.Contains(t, body, `>Valider</button>`)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a valid form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permWrite)

		rec := s.post(t, "/webui/users/new?ret=sort%3Dservers%26dir%3Ddesc", url.Values{
			"login": {" newbie "}, "password": {"s3cret-pass"}, "confirm": {"s3cret-pass"},
			"servers": {"r", "w"}, "partners": {"r"}, "transfers": {"r", "d"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/users?dir=desc&done=created&name=newbie&sort=servers", rec.Header().Get("Location"))

		var user model.User
		require.NoError(t, s.db.Get(&user, "username=?", "newbie").Run())
		assert.Equal(t, model.PermServersRead|model.PermServersWrite|model.PermPartnersRead|model.PermTransfersRead,
			user.Permissions, "the delete right on the transfers is never set")
		assert.True(t, utils.CheckPassword(utils.BcryptRounds, user.PasswordHash, "s3cret-pass"), "the password opens")

		body := s.get(t, "/webui/users?done=created&name=newbie").Body.String()
		assert.Contains(t, body, "Utilisateur « newbie » créé")
	})

	t.Run("Given an invalid form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permWrite)
		s.fixtures(t)

		rec := s.post(t, "/webui/users/new", url.Values{"login": {""}, "password": {"abc"}, "confirm": {"abd"}, "rules": {"r"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<div class="w-banner w-banner--ko" role="alert">`)
		assert.Contains(t, body, "L&#39;utilisateur n&#39;a pas été enregistré.")
		assert.Contains(t, body, `<details class="w-card w-card--fold" open>`, "the refused rights card opens, its kept state set aside (#807)")
		assert.NotContains(t, body, `data-key="users-form-rights"`)
		assert.Contains(t, body, "Ce champ est obligatoire.")
		assert.Contains(t, body, "Les deux saisies du mot de passe diffèrent.")
		assert.NotContains(t, body, "abc", "no password is rendered back")
		assert.Contains(t, body, `name="rules" value="r" checked`, "the rights are kept")
		assert.EqualValues(t, 4, s.count(t), "nothing created")

		rec = s.post(t, "/webui/users/new", url.Values{"login": {"x"}, "password": {""}, "confirm": {""}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez le mot de passe.")

		long := strings.Repeat("x", maxPasswordBytes+1)
		rec = s.post(t, "/webui/users/new", url.Values{"login": {"x"}, "password": {long}, "confirm": {long}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "72 caractères au plus.")
		assert.NotContains(t, rec.Body.String(), long)

		rec = s.post(t, "/webui/users/new", url.Values{"login": {"viewer"}, "password": {"pw"}, "confirm": {"pw"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre utilisateur porte déjà cet identifiant.")
		assert.EqualValues(t, 4, s.count(t), "nothing created")
	})
}

func TestEditUser(t *testing.T) {
	t.Parallel()

	t.Run("Given the form of the viewer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		body := s.get(t, path(s.viewer, "/edit")+"?ret=page%3D2").Body.String()
		assert.Contains(t, body, "<title>Modifier « viewer » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `id="login" name="login" value="viewer" required`)
		assert.Contains(t, body, `id="password" name="password" type="password" autocomplete="new-password"`, "the password is optional")
		assert.Contains(t, body, "Vide : inchangé. 72 caractères au plus.")
		assert.Contains(t, body, `name="servers" value="r" checked`)
		assert.Contains(t, body, `name="servers" value="w">`, "not checked")
		assert.Equal(t, 1, strings.Count(body, " checked"), "one right held")
		assert.NotContains(t, body, "C&#39;est votre compte")
		assert.Contains(t, body, `href="`+path(s.viewer, "")+`?ret=page%3D2">`, "cancel goes back to the user")
	})

	t.Run("Given the signed-in user's own form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		body := s.get(t, path(s.tester, "/edit")).Body.String()
		assert.Contains(t, body, "C&#39;est votre compte : vos droits s&#39;appliquent dès l&#39;enregistrement.")
		assert.Contains(t, body, "Vous êtes le dernier administrateur")

		// With another administrator, the warning is shorter.
		require.NoError(t, s.db.Insert(&model.User{Username: "root", PasswordHash: hashOf(t, "root-pw"), Permissions: permAll}).Run())
		body = s.get(t, path(s.tester, "/edit")).Body.String()
		assert.Contains(t, body, "C&#39;est votre compte")
		assert.NotContains(t, body, "dernier administrateur")
	})

	t.Run("Given a saved edition without password", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)
		before := s.viewer.PasswordHash

		rec := s.post(t, path(s.viewer, "/edit")+"?ret=q%3Dvi", url.Values{"login": {"viewer2"}, "partners": {"r"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, path(s.viewer, "")+"?done=updated&name=viewer2&ret=q%3Dvi", rec.Header().Get("Location"),
			"an edited user comes back to their sheet")

		saved := s.user(t, s.viewer.ID)
		assert.Equal(t, "viewer2", saved.Username)
		assert.Equal(t, model.PermPartnersRead, saved.Permissions)
		assert.Equal(t, before, saved.PasswordHash, "an empty password keeps the current one")

		body := s.get(t, path(s.viewer, "")+"?done=updated&name=viewer2").Body.String()
		assert.Contains(t, body, "Utilisateur « viewer2 » enregistré")
	})

	t.Run("Given a saved edition with a new password", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		rec := s.post(t, path(s.ops, "/edit"), url.Values{"login": {"ops"}, "password": {"changed-pw"}, "confirm": {"changed-pw"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		saved := s.user(t, s.ops.ID)
		assert.True(t, utils.CheckPassword(utils.BcryptRounds, saved.PasswordHash, "changed-pw"), "the new password opens")
		assert.False(t, utils.CheckPassword(utils.BcryptRounds, saved.PasswordHash, "ops-pw"), "the old one no longer does")
		assert.Zero(t, saved.Permissions, "no checkbox posted: no right left")
	})

	t.Run("Given the signed-in user dropping their own rights", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)

		// Keeping the read right: back to the sheet.
		rec := s.post(t, path(s.tester, "/edit"), url.Values{"login": {"tester"}, "users": {"r"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, path(s.tester, "")+"?done=updated&name=tester", rec.Header().Get("Location"))
		assert.Equal(t, model.PermUsersRead, s.user(t, s.tester.ID).Permissions)

		// Dropping it: the sheet would be refused, "My account" takes over.
		rec = s.post(t, path(s.tester, "/edit"), url.Values{"login": {"tester"}, "servers": {"r"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/account?done=account", rec.Header().Get("Location"))
		assert.Equal(t, model.PermServersRead, s.user(t, s.tester.ID).Permissions)
	})

	t.Run("Given a login already taken", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		rec := s.post(t, path(s.ops, "/edit"), url.Values{"login": {"viewer"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre utilisateur porte déjà cet identifiant.")
		assert.Equal(t, "ops", s.user(t, s.ops.ID).Username, "nothing changed")
	})
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	t.Run("Given the confirmation page and the deletion", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		body := s.get(t, path(s.viewer, "/delete")+"?ret=q%3Dvi").Body.String()
		assert.Contains(t, body, "<title>Supprimer l&#39;utilisateur « viewer » ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Il ne pourra plus se connecter à l&#39;interface, à l&#39;API ni à la ligne de commande")
		assert.Contains(t, body, `<dt>Identifiant</dt><dd><code class="w-code">viewer</code></dd>`)
		assert.Contains(t, body, `<strong class="rights-domain">Serveurs et clients locaux</strong> <span class="rights-granted">Lecture</span>`,
			"the rights are recalled")
		assert.Contains(t, body, `action="`+path(s.viewer, "/delete")+`?ret=q%3Dvi"`)
		assert.Contains(t, body, ">Supprimer</button>")

		rec := s.post(t, path(s.viewer, "/delete")+"?ret=q%3Dvi", nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/users?done=deleted&name=viewer&q=vi", rec.Header().Get("Location"))

		_, err := internal.GetUserByID(s.db, s.viewer.ID)
		assert.True(t, database.IsNotFound(err))
	})

	t.Run("Given the signed-in user's own account", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		s.fixtures(t)

		rec := s.get(t, path(s.tester, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Cet utilisateur ne peut pas être supprimé.")
		assert.Contains(t, rec.Body.String(), "Vous ne pouvez pas supprimer votre propre compte.")
		assert.NotContains(t, rec.Body.String(), ">Supprimer</button>")

		rec = s.post(t, path(s.tester, "/delete"), nil)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.EqualValues(t, 4, s.count(t), "still there")
	})

	t.Run("Given the last administrator", func(t *testing.T) {
		t.Parallel()

		// The tester may delete users but not manage them: the only
		// administrator is another user.
		s := newSuite(t, permRead|model.PermUsersDelete)
		s.fixtures(t)
		admin := &model.User{Username: "root", PasswordHash: hashOf(t, "root-pw"), Permissions: permWrite}
		require.NoError(t, s.db.Insert(admin).Run())

		rec := s.get(t, path(admin, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "C&#39;est le dernier à pouvoir gérer les utilisateurs")
		assert.NotContains(t, rec.Body.String(), ">Supprimer</button>")

		rec = s.post(t, path(admin, "/delete"), nil)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		_, err := internal.GetUserByID(s.db, admin.ID)
		require.NoError(t, err, "still there")

		// A second administrator: the first one can go.
		require.NoError(t, s.db.Insert(&model.User{Username: "root2", PasswordHash: hashOf(t, "pw"), Permissions: permWrite}).Run())
		rec = s.post(t, path(admin, "/delete"), nil)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		_, err = internal.GetUserByID(s.db, admin.ID)
		assert.True(t, database.IsNotFound(err))
	})

	t.Run("Given an unknown user", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, permAll)
		assert.Equal(t, http.StatusNotFound, s.post(t, "/webui/users/999/delete", nil).Code)
	})
}

// No password, nor its hash, ever reaches a page: neither the list, nor the
// sheet, nor the forms, refused or not.
func TestNoSecret(t *testing.T) {
	t.Parallel()

	s := newSuite(t, permAll)
	s.fixtures(t)

	for _, target := range []string{"/webui/users", path(s.viewer, ""), path(s.viewer, "/edit"), path(s.viewer, "/delete")} {
		body := s.get(t, target).Body.String()
		assert.NotContains(t, body, "viewer-pw", target)
		assert.NotContains(t, body, s.viewer.PasswordHash, target)
		assert.NotContains(t, body, "$2a$", target)
	}

	body := s.post(t, path(s.viewer, "/edit"), url.Values{"login": {""}, "password": {"typed-pw"}, "confirm": {"other"}}).Body.String()
	assert.NotContains(t, body, "typed-pw")
	assert.NotContains(t, body, "other")
	assert.NotContains(t, body, "$2a$")
}

func TestFilterMatches(t *testing.T) {
	t.Parallel()

	servers := model.PermServersRead | model.PermServersWrite
	all := model.PermAll

	for _, c := range []struct {
		domain, right string
		perms         model.PermsMask
		want          bool
	}{
		{"", "", 0, true},
		{"servers", "", servers, true},
		{"servers", "", model.PermPartnersRead, false},
		{"servers", "read", servers, true},
		{"servers", "delete", servers, false},
		{"servers", "delete", all, true},
		{"transfers", "delete", all, false},
		{"", "delete", servers, false},
		{"", "delete", model.PermRulesDelete, true},
		{"", "none", 0, true},
		{"", "none", model.PermRulesRead, false},
		{"rules", "none", servers, true},
		{"rules", "none", model.PermRulesRead, false},
		{"bogus", "bogus", 0, true},
	} {
		f := parseFilter(url.Values{domainParam: {c.domain}, rightParam: {c.right}})
		assert.Equalf(t, c.want, f.(*filter).Matches(c.perms), "%s %s %b", c.domain, c.right, c.perms)
	}
}
