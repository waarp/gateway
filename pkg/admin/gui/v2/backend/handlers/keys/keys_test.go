package keys

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.waarp.fr/lib/gui"
	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermAdminRead
	permWrite = model.PermAdminRead | model.PermAdminWrite
	permAll   = model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	day = 24 * time.Hour

	hmacSecret = "dW4tc2VjcmV0LWhtYWMtZGUtdGVzdA==" // in base64, as the tasks read it (#736)
)

// aesKey is a key of n bytes, in base64 as the tasks read it.
func aesKey(n int) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, n))
}

// pgpPair is a PGP key made for the tests: the private key and the public
// one, armoured.
type pgpPair struct {
	Private, Public string
	key             *pgp.Key
}

// newPGP makes a PGP key for an identity, created at the given time, valid
// for the given lifetime (0: no expiry).
func newPGP(t *testing.T, name string, created time.Time, lifetime time.Duration) *pgpPair {
	t.Helper()

	key, err := pgp.PGP().KeyGeneration().
		AddUserId(name, strings.ToLower(name)+"@example.org").
		GenerationTime(created.Unix()).
		Lifetime(int32(lifetime / time.Second)).
		New().GenerateKey()
	require.NoError(t, err)

	private, err := key.Armor()
	require.NoError(t, err)

	public, err := key.GetArmoredPublicKey()
	require.NoError(t, err)

	return &pgpPair{Private: private, Public: public, key: key}
}

// locked is the private key protected by a passphrase.
func (p *pgpPair) locked(t *testing.T) string {
	t.Helper()

	key, err := pgp.PGP().LockKey(p.key, []byte("phrase secrète"))
	require.NoError(t, err)

	armored, err := key.Armor()
	require.NoError(t, err)

	return armored
}

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router
	now    time.Time

	partner                 *pgpPair
	aes, hmac, pub, private *model.CryptoKey
	send, receive, other    *model.Rule
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter(), now: time.Now()}
	h := New(s.db, testhelpers.GetTestLogger(t))
	h.now = func() time.Time { return s.now }

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/keys").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/keys/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/keys/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/keys/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/keys/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/keys/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/keys/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/keys/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures inserts four keys (AES-128, HMAC, the public key of a partner
// expiring in 20 days, a private key without expiry) and three rules: the
// "flux-paie" rule in both directions, whose tasks encrypt with the AES key
// on sending and decrypt with it on receiving, the sending one also signing
// with HMAC; the "flux-pgp" rule, which encrypts for the partner and signs
// with the private key in one task.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.partner = newPGP(t, "Partner", s.now.Add(-20*day), 40*day)
	own := newPGP(t, "Gateway", s.now.Add(-day), 0)

	s.aes = &model.CryptoKey{Name: "aes-paie", Type: model.CryptoKeyTypeAES, Key: aesKey(aes128)}
	s.hmac = &model.CryptoKey{Name: "hmac-paie", Type: model.CryptoKeyTypeHMAC, Key: hmacSecret}
	s.pub = &model.CryptoKey{Name: "pgp-partner", Type: model.CryptoKeyTypePGPPublic, Key: s.partner.Public}
	s.private = &model.CryptoKey{Name: "pgp-gateway", Type: model.CryptoKeyTypePGPPrivate, Key: own.Private}

	for _, k := range []*model.CryptoKey{s.aes, s.hmac, s.pub, s.private} {
		require.NoError(t, s.db.Insert(k).Run())
	}

	s.send = &model.Rule{Name: "flux-paie", IsSend: true, Path: "/paie-send"}
	s.receive = &model.Rule{Name: "flux-paie", IsSend: false, Path: "/paie-receive"}
	s.other = &model.Rule{Name: "flux-pgp", IsSend: true, Path: "/pgp"}

	for _, r := range []*model.Rule{s.send, s.receive, s.other} {
		require.NoError(t, s.db.Insert(r).Run())
	}

	for _, task := range []*model.Task{
		{
			RuleID: s.send.ID, Chain: model.ChainPre, Rank: 0, Type: tasks.Encrypt,
			Args: map[string]string{"keyName": "aes-paie", "method": tasks.EncryptMethodAESCFB},
		},
		{
			RuleID: s.send.ID, Chain: model.ChainPre, Rank: 1, Type: tasks.Sign,
			Args: map[string]string{"keyName": "hmac-paie", "method": tasks.SignMethodHMACSHA256},
		},
		{
			RuleID: s.send.ID, Chain: model.ChainError, Rank: 0, Type: tasks.Encrypt,
			Args: map[string]string{"keyName": "aes-paie", "method": tasks.EncryptMethodAESCFB},
		},
		{
			RuleID: s.receive.ID, Chain: model.ChainPost, Rank: 0, Type: tasks.Decrypt,
			Args: map[string]string{"keyName": "aes-paie", "method": tasks.EncryptMethodAESCFB},
		},
		{RuleID: s.other.ID, Chain: model.ChainPre, Rank: 0, Type: tasks.EncryptAndSign, Args: map[string]string{
			"encryptKeyName": "pgp-partner", "signKeyName": "pgp-gateway", "method": tasks.EncryptSignMethodPGP,
		}},
	} {
		require.NoError(t, s.db.Insert(task).Run())
	}
}

// insert stores a key as a database may hold it: through the model when it
// accepts the value, else, for a value the model refuses since #736 and #737
// (an AES key of text, a locked private key, kept from before), with a valid
// value of its type, then its value written over, encrypted as the model
// writes it.
func (s *suite) insert(t *testing.T, name, typ, value string) *model.CryptoKey {
	t.Helper()

	k := &model.CryptoKey{Name: name, Type: typ, Key: value}
	if err := s.db.Insert(k).Run(); err == nil {
		return k
	}

	valid := map[string]string{
		model.CryptoKeyTypeAES: aesKey(aes128), model.CryptoKeyTypeHMAC: hmacSecret,
		model.CryptoKeyTypePGPPublic:  newPGP(t, "Valid", time.Now(), 0).Public,
		model.CryptoKeyTypePGPPrivate: newPGP(t, "Valid", time.Now(), 0).Private,
	}
	k = &model.CryptoKey{Name: name, Type: typ, Key: valid[typ]}
	require.NoError(t, s.db.Insert(k).Run())

	stored, err := s.db.Encrypt(value)
	require.NoError(t, err)
	require.NoError(t, s.db.Exec("UPDATE "+model.TableCryptoKeys+" SET value=? WHERE id=?", stored, k.ID))
	k.Key = value

	return k
}

func (s *suite) request(t *testing.T, perms model.PermsMask, method, target string, body *bytes.Buffer, contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")

	var req *http.Request
	if body != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, body)
		req.Header.Set("Content-Type", contentType)
	} else {
		req = httptest.NewRequestWithContext(ctx, method, target, nil)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *suite) get(t *testing.T, perms model.PermsMask, target string) *httptest.ResponseRecorder {
	t.Helper()

	return s.request(t, perms, http.MethodGet, target, nil, "")
}

func (s *suite) post(t *testing.T, perms model.PermsMask, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	return s.request(t, perms, http.MethodPost, target, bytes.NewBufferString(form.Encode()), "application/x-www-form-urlencoded")
}

// upload posts a multipart form: the fields, then a file in the given field.
func (s *suite) upload(t *testing.T, target string, fields map[string]string, fileField, content string,
) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}

	part, err := w.CreateFormFile(fileField, "key.asc")
	require.NoError(t, err)

	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return s.request(t, permWrite, http.MethodPost, target, body, w.FormDataContentType())
}

func (s *suite) key(t *testing.T, name string) *model.CryptoKey {
	t.Helper()

	var k model.CryptoKey
	require.NoError(t, s.db.Get(&k, "name=?", name).Run())

	return &k
}

func path(k *model.CryptoKey, suffix string) string {
	return "/webui/keys/" + strconv.FormatInt(k.ID, 10) + suffix
}

func tasksURL(r *model.Rule) string {
	return "/webui/rules/" + strconv.FormatInt(r.ID, 10) + "/tasks"
}

// indexes asserts that the markers appear in the body in the given order.
func indexes(t *testing.T, body string, markers ...string) {
	t.Helper()

	from := 0

	for _, marker := range markers {
		pos := strings.Index(body[from:], marker)
		require.NotEqualf(t, -1, pos, "%q not found in order in the page", marker)
		from += pos + len(marker)
	}
}

func rows(body string) int { return strings.Count(body, "<tr>") - 1 }

// assertNoSecret checks that no secret value of the fixtures is in a page.
func (s *suite) assertNoSecret(t *testing.T, body string) {
	t.Helper()

	assert.NotContains(t, body, s.aes.Key)
	assert.NotContains(t, body, hmacSecret)
	assert.NotContains(t, body, ">-----BEGIN PGP PRIVATE KEY BLOCK")
	assert.NotContains(t, body, "PRIVATE KEY BLOCK-----\n")
}

// TestListFragment: the region of the list alone, for the live list of the kit (#766):
// the elements of the part "list", with the ids of the page, no shell; the
// page itself declares the live region and its filters form.
func TestListFragment(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permAll, "/webui/keys?fragment=list")
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.NotContains(t, body, "<html", "no shell around a fragment")
	assert.Contains(t, body, `<span class="w-toolbar__count" id="keys-count" data-live-part="list" aria-live="polite">`)
	assert.Contains(t, body, `<div id="keys-list" data-live-part="list" data-live-paged>`)

	body = s.get(t, permAll, "/webui/keys").Body.String()
	assert.Contains(t, body, `<div data-live data-live-period="0">`, "read on the user's actions only")
	assert.Contains(t, body, `data-autosubmit data-live-filter>`, "the filters apply in place")
	assert.Contains(t, body, `<template data-live-error>`)
	assert.Contains(t, body, `live.js`)
}

func TestKeyInfo(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("AES keys, read as the tasks read them", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, keyInfo{Readable: true, Bits: 128}, aesInfo(aesKey(16)))
		assert.Equal(t, keyInfo{Readable: true, Bits: 192}, aesInfo(aesKey(24)))
		assert.Equal(t, keyInfo{Readable: true, Bits: 256}, aesInfo(aesKey(32)))
		assert.Equal(t, keyInfo{Cause: causeAES}, aesInfo("0123456789abcdef"), "16 characters decode to 12 bytes")
		assert.Equal(t, keyInfo{Cause: causeAES}, aesInfo("not base64!"))
	})

	t.Run("A PGP key with an identity and an expiry", func(t *testing.T) {
		t.Parallel()

		p := newPGP(t, "Partner", now.Add(-20*day), 40*day)
		info := pgpInfo(model.CryptoKeyTypePGPPublic, p.Public, now)

		require.True(t, info.Readable)
		assert.Equal(t, []string{"Partner <partner@example.org>"}, info.Identities)
		assert.Regexp(t, `^[0-9A-F]{4}( [0-9A-F]{4}){9}$`, info.Fingerprint, "a v4 fingerprint by groups of four")
		assert.Equal(t, strings.ReplaceAll(info.Fingerprint, " ", "")[24:], info.KeyID, "the key ID ends the fingerprint")
		assert.NotEmpty(t, info.Algorithm)
		assert.Equal(t, now.Add(-20*day).Unix(), info.Created.Unix())
		assert.Equal(t, now.Add(20*day).Unix(), info.Expires.Unix())
		assert.Equal(t, strings.TrimSpace(p.Public), info.Public)
		assert.False(t, info.Expired)

		state := info.state(echo, now)
		assert.Equal(t, stateWarn, state.State, "expires within 30 days")
		assert.Equal(t, "keys.state.warn date "+info.Expires.Format("auth.date"), state.Label)

		assert.Equal(t, stateOK, info.state(echo, now.Add(-30*day)).State)
	})

	t.Run("PGP keys the tasks cannot use", func(t *testing.T) {
		t.Parallel()

		p := newPGP(t, "Partner", now.Add(-20*day), 10*day)

		expired := pgpInfo(model.CryptoKeyTypePGPPrivate, p.Private, now)
		assert.True(t, expired.Readable && expired.Expired)
		assert.Equal(t, stateKO, expired.state(echo, now).State)
		assert.Empty(t, expired.Public, "a private key is never rendered")

		assert.Equal(t, causeLocked, pgpInfo(model.CryptoKeyTypePGPPrivate, p.locked(t), now).Cause)
		assert.Equal(t, causeNotPublic, pgpInfo(model.CryptoKeyTypePGPPublic, p.Private, now).Cause)
		assert.Equal(t, causeNotPrivate, pgpInfo(model.CryptoKeyTypePGPPrivate, p.Public, now).Cause)
		assert.Equal(t, causePGP, pgpInfo(model.CryptoKeyTypePGPPublic, "garbage", now).Cause)

		unusable := keyInfo{Cause: causeLocked}
		assert.Equal(t, gui.Status{State: stateKO, Label: "keys.state.unusable cause keys.cause.locked"},
			unusable.state(echo, now))
	})

	t.Run("HMAC keys and unknown types", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, keyInfo{Readable: true}, infoOf(&model.CryptoKey{Type: model.CryptoKeyTypeHMAC, Key: hmacSecret}, now))
		assert.Equal(t, keyInfo{Cause: causeHMAC}, infoOf(&model.CryptoKey{Type: model.CryptoKeyTypeHMAC, Key: "pas en base64"}, now),
			"the tasks read an HMAC secret in base64 (#736)")
		assert.Equal(t, keyInfo{Cause: causeEmpty}, infoOf(&model.CryptoKey{Type: model.CryptoKeyTypeHMAC}, now))
		assert.Equal(t, keyInfo{Cause: causeType}, infoOf(&model.CryptoKey{Type: "DES"}, now))
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.insert(t, "aes-texte", model.CryptoKeyTypeAES, "0123456789abcdef")

		rec := s.get(t, permAll, "/webui/keys")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Clés cryptographiques · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/keys" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/keys/new">`)
		assert.Equal(t, 4, strings.Count(body, `type="checkbox" name="type"`), "four types")
		assert.Contains(t, body, `aria-live="polite">5 clés</span>`)

		assert.Equal(t, 5, rows(body))
		indexes(t, body, "<tbody>", ">aes-paie<", ">aes-texte<", ">hmac-paie<", ">pgp-gateway<", ">pgp-partner<", "</tbody>")
		assert.Contains(t, body, `>aes-paie</a></td><td class="w-nowrap">AES</td><td class="keys-detail"><span>AES-128</span> </td>`+
			`<td class="keys-state"><span class="w-status w-status--ok">`)
		assert.Contains(t, body, `>aes-texte</a></td><td class="w-nowrap">AES</td><td class="keys-detail"><span>Illisible</span> </td>`)
		assert.Contains(t, body, "Inutilisable : la valeur n&#39;est pas une clé AES en base64")
		assert.Contains(t, body, `<td class="keys-detail"><span>Secret défini</span> </td>`)
		assert.Regexp(t, `<td class="keys-detail"><span>Partner &lt;partner@example.org&gt;</span> `+
			`<span class="keys-code w-mono">[0-9A-F]{8}</span></td>`, body, "the identity, then the short key ID")
		assert.Contains(t, body, "Expire le "+s.now.Add(20*day).Format("02/01/2006"))
		assert.Contains(t, body, `<td class="w-nowrap">2 règles</td>`, "the AES key: both directions of flux-paie")
		assert.Contains(t, body, `<td class="w-nowrap">1 règle</td>`)
		assert.Contains(t, body, `<td class="w-nowrap">Aucune règle</td>`)
		indexes(t, body, ">aes-paie</a>", "Voir la fiche", "Modifier la clé", "Supprimer la clé", ">aes-texte</a>")
		s.assertNoSecret(t, body)
	})

	t.Run("Given a type filter, a search and a sort", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/keys?type="+url.QueryEscape(model.CryptoKeyTypePGPPublic)).Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, ">pgp-partner</a>")
		assert.Contains(t, body, `name="type" value="PGP-PUBLIC" checked`)

		body = s.get(t, permRead, "/webui/keys?q=PGP").Body.String()
		assert.Equal(t, 2, rows(body))

		body = s.get(t, permRead, "/webui/keys?sort=type&dir=desc").Body.String()
		indexes(t, body, ">pgp-partner<", ">pgp-gateway<", ">hmac-paie<", ">aes-paie<")

		body = s.get(t, permRead, "/webui/keys?q=zz").Body.String()
		assert.Contains(t, body, "Aucune clé ne commence par « zz »")
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/keys").Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action without the write right")
		assert.NotContains(t, body, "Modifier la clé")
		assert.NotContains(t, body, "Supprimer la clé")
		assert.Contains(t, body, "Voir la fiche")
	})

	t.Run("Given no key at all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permAll, "/webui/keys").Body.String()
		assert.Contains(t, body, "Aucune clé cryptographique pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/keys/new"`)
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Given the AES key the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, path(s.aes, ""))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Clé « aes-paie »")
		assert.Contains(t, body, `AES <code class="w-code">AES</code>`)
		assert.Contains(t, body, "Chiffre et déchiffre les fichiers")
		assert.Contains(t, body, "Définie, jamais affichée")
		assert.Contains(t, body, "128 bits")
		indexes(t, body, "Utilisée par", `<ul class="ruleuse-list" aria-label="Règles dont les tâches emploient la clé">`,
			`<span class="ruleuse-direction">Envoi</span> <a class="w-link ruleuse-rule" href="`+tasksURL(s.send)+`">flux-paie</a>`,
			`<span class="ruleuse-direction">Réception</span> <a class="w-link ruleuse-rule" href="`+tasksURL(s.receive)+`">flux-paie</a>`)
		assert.Contains(t, body, `href="/webui/static_v2/ruleuse/ruleuse.css"`, "the cards of the rules")
		assert.NotContains(t, body, "<table", "no table in a card")
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-key="keys-value">`, "the cards after the first fold (#807)")
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-key="keys-uses">`)
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="`+path(s.aes, "/edit")+`"`)
		s.assertNoSecret(t, body)
	})

	t.Run("Given the PGP keys", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, path(s.pub, "")).Body.String()
		assert.Contains(t, body, "<li>Partner &lt;partner@example.org&gt;</li>")
		assert.Contains(t, body, "-----BEGIN PGP PUBLIC KEY BLOCK-----", "the public key in full, folded")
		assert.Contains(t, body, s.now.Add(20*day).Format("02/01/2006"))
		assert.Contains(t, body, `href="`+tasksURL(s.other)+`">flux-pgp</a>`)
		assert.NotContains(t, body, "w-btn--primary", "no edition without the write right")

		body = s.get(t, permRead, path(s.private, "")).Body.String()
		assert.Contains(t, body, "<li>Gateway &lt;gateway@example.org&gt;</li>")
		assert.Contains(t, body, "Définie, jamais affichée")
		assert.Contains(t, body, "<dd>Aucune</dd>", "no expiry")
		assert.NotContains(t, body, "-----BEGIN PGP", "neither the private key nor its public part")
		s.assertNoSecret(t, body)
	})

	t.Run("Given a key no rule uses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		k := s.insert(t, "pgp-verrou", model.CryptoKeyTypePGPPrivate,
			newPGP(t, "Locked", time.Now(), 0).locked(t))

		body := s.get(t, permRead, path(k, "")).Body.String()
		assert.Contains(t, body, "Inutilisable : clé privée protégée par une phrase secrète")
		assert.Contains(t, body, "Aucune règle ne l&#39;emploie.")
	})

	t.Run("Given an unknown key", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permRead, "/webui/keys/999")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Clé introuvable")
	})
}

func TestNewKey(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permWrite, "/webui/keys/new").Body.String()
		assert.Contains(t, body, `enctype="multipart/form-data"`)
		assert.Contains(t, body, `<select class="w-select" id="type" name="type" required data-switch aria-describedby="type-h">`)
		assert.Contains(t, body, `<option value="PGP-PRIVATE">PGP privée</option>`)

		for _, k := range kinds {
			assert.Contains(t, body, `data-case="`+k.Key+`"`)
		}

		assert.Contains(t, body, `id="value_aes" name="value_aes" type="password"`)
		assert.Contains(t, body, `autocomplete="new-password"`)
		assert.Contains(t, body, "24 caractères pour AES-128, 32 pour AES-192, 44 pour AES-256")
		assert.Contains(t, body, `name="value_pgp_public_file"`)
		assert.Contains(t, body, `placeholder="-----BEGIN PGP PUBLIC KEY BLOCK-----"`)
		assert.NotRegexp(t, `<(input|select|textarea)[^>]* id="key"`, body, "no field id collides with an icon of the kit")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given keys of every type", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		p := newPGP(t, "Partner", time.Now(), 0)

		rec := s.post(t, permWrite, "/webui/keys/new", url.Values{
			"name": {" aes-192 "}, "type": {model.CryptoKeyTypeAES}, "value_aes": {" " + aesKey(aes192) + "\n"},
			"value_hmac": {"ignored"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), "done=created")
		assert.Equal(t, aesKey(aes192), s.key(t, "aes-192").Key)

		rec = s.post(t, permWrite, "/webui/keys/new",
			url.Values{"name": {"hmac"}, "type": {model.CryptoKeyTypeHMAC}, "value_hmac": {" " + hmacSecret + "\n"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, hmacSecret, s.key(t, "hmac").Key, "an HMAC secret in base64, the spaces around left out")

		rec = s.upload(t, "/webui/keys/new", map[string]string{"name": "pub", "type": model.CryptoKeyTypePGPPublic},
			"value_pgp_public_file", strings.ReplaceAll(p.Public, "\n", "\r\n"))
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, strings.TrimSpace(p.Public), s.key(t, "pub").Key)

		rec = s.post(t, permWrite, "/webui/keys/new",
			url.Values{"name": {"priv"}, "type": {model.CryptoKeyTypePGPPrivate}, "value_pgp_private_text": {p.Private}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, model.CryptoKeyTypePGPPrivate, s.key(t, "priv").Type)
	})

	t.Run("Given forms the screen or the model refuses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		p := newPGP(t, "Partner", time.Now(), 0)
		refused := func(form url.Values, want string) string {
			t.Helper()

			rec := s.post(t, permWrite, "/webui/keys/new", form)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

			body := rec.Body.String()
			assert.Contains(t, body, "La clé n&#39;a pas été enregistrée.")
			assert.Contains(t, body, want)

			return body
		}

		refused(url.Values{"name": {""}, "type": {""}}, "Choisissez un type.")
		refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypeHMAC}}, "Ce champ est obligatoire.")

		body := refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypeAES}, "value_aes": {"0123456789abcdef"}},
			"Indiquez la clé en base64")
		assert.NotContains(t, body, "0123456789abcdef", "a secret typed is never rendered back")

		refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypeAES}, "value_aes": {aesKey(20)}},
			"Indiquez la clé en base64")
		refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypeHMAC}, "value_hmac": {"pas en base64"}},
			"Indiquez le secret en base64.")
		refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypePGPPrivate}, "value_pgp_private_text": {p.locked(t)}},
			"Cette clé privée est protégée par une phrase secrète")
		refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypePGPPrivate}, "value_pgp_private_text": {p.Public}},
			"Cette clé est une clé publique")

		body = refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypePGPPublic}, "value_pgp_public_text": {p.Private}},
			"Cette clé est une clé privée")
		assert.NotContains(t, body, ">-----BEGIN PGP PRIVATE KEY BLOCK", "a private key pasted is never rendered back")

		body = refused(url.Values{"name": {"x"}, "type": {model.CryptoKeyTypePGPPublic}, "value_pgp_public_text": {"garbage"}},
			"Ce contenu n&#39;est pas une clé PGP lisible")
		assert.Contains(t, body, ">garbage</textarea>", "a public armour pasted comes back")

		refused(url.Values{"name": {"aes-paie"}, "type": {model.CryptoKeyTypeHMAC}, "value_hmac": {hmacSecret}},
			"Une autre clé porte déjà ce nom.")

		count, err := s.db.Count(&model.CryptoKey{}).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 4, count, "nothing created")
	})
}

func TestEditKey(t *testing.T) {
	t.Parallel()

	t.Run("Given a key no rule uses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		k := s.insert(t, "hmac-libre", model.CryptoKeyTypeHMAC, hmacSecret)

		body := s.get(t, permWrite, path(k, "/edit")).Body.String()
		assert.Contains(t, body, `id="name" name="name" value="hmac-libre"`)
		assert.Contains(t, body, `<option value="HMAC" selected>`)
		assert.Contains(t, body, "Laissez vide pour conserver le secret en place.")
		assert.NotContains(t, body, hmacSecret)

		rec := s.post(t, permWrite, path(k, "/edit"), url.Values{"name": {"hmac-renommee"}, "type": {model.CryptoKeyTypeHMAC}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), path(k, "")+"?done=updated")
		assert.Equal(t, hmacSecret, s.key(t, "hmac-renommee").Key, "an empty secret keeps the stored one")

		rec = s.post(t, permWrite, path(k, "/edit"), url.Values{"name": {"hmac-renommee"}, "type": {model.CryptoKeyTypeAES}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce champ est obligatoire.", "a new type asks for a new value")
		assert.Equal(t, model.CryptoKeyTypeHMAC, s.key(t, "hmac-renommee").Type, "nothing changed")
	})

	t.Run("Given a key the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.aes, "/edit")).Body.String()
		assert.Contains(t, body, "Employée par 2 règles : le nom et le type se conservent")
		assert.NotContains(t, body, `name="name"`, "the name is shown in reading")
		assert.NotContains(t, body, `name="type"`)
		assert.Contains(t, body, `name="value_aes"`)
		assert.NotContains(t, body, `name="value_hmac"`, "the value of its type only")
		assert.Contains(t, body, "AES-128", "what is in place")
		assert.NotContains(t, body, "champs obligatoires", "nothing is required")

		rec := s.post(t, permWrite, path(s.aes, "/edit"), url.Values{
			"name": {"renommee"}, "type": {model.CryptoKeyTypeHMAC}, "value_aes": {aesKey(aes192)},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		k := s.key(t, "aes-paie")
		assert.Equal(t, model.CryptoKeyTypeAES, k.Type, "the name and the type are kept")
		assert.Equal(t, aesKey(aes192), k.Key, "the value is replaced")
	})
}

func TestDeleteKey(t *testing.T) {
	t.Parallel()

	t.Run("Given a key no rule uses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		k := s.insert(t, "hmac-libre", model.CryptoKeyTypeHMAC, hmacSecret)

		body := s.get(t, permAll, path(k, "/delete")).Body.String()
		assert.Contains(t, body, "Supprimer la clé « hmac-libre » ?")
		assert.Contains(t, body, "Aucune règle ne l&#39;emploie. Elle sera supprimée définitivement.")
		assert.Contains(t, body, ">Supprimer</button>")

		rec := s.post(t, permAll, path(k, "/delete"), url.Values{})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=deleted")

		count, err := s.db.Count(&model.CryptoKey{}).Where("name=?", "hmac-libre").Run()
		require.NoError(t, err)
		assert.Zero(t, count)
	})

	t.Run("Given a key the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, path(s.aes, "/delete")).Body.String()
		assert.Contains(t, body, "Cette clé ne peut pas être supprimée.")
		assert.Contains(t, body, "Elle est employée par 2 règles : changez d&#39;abord la clé de leurs tâches")
		assert.Contains(t, body, `href="`+tasksURL(s.send)+`">flux-paie</a>`)
		assert.NotContains(t, body, ">Supprimer</button>", "no deletion")
		assert.Contains(t, body, ">Fermer</a>")

		rec := s.post(t, permAll, path(s.aes, "/delete"), url.Values{})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, s.aes.Key, s.key(t, "aes-paie").Key, "the key stays")
	})
}

// echo is a translator that writes the key and its arguments.
func echo(key string, args ...any) string {
	return strings.TrimSpace(fmt.Sprintln(append([]any{key}, args...)...))
}
