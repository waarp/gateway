package dashboard

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// at is a moment of the days of the fixtures, in the time zone of the gateway.
func at(day, hour, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, time.Local)
}

// certPEM makes a self-signed certificate for a name, valid until notAfter,
// and its key, both in PEM.
func certPEM(t *testing.T, name string, notAfter time.Time) (cert, key string) {
	t.Helper()

	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, BasicConstraintsValid: true, IsCA: true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(private)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
}

// sampleServices is a lister with a server in error.
func sampleServices() (core, servers, clients []internal.Service) {
	core = []internal.Service{
		{Name: "Database", State: utils.StateRunning.String()},
		{Name: "Controller", State: utils.StateRunning.String()},
	}
	servers = []internal.Service{
		{Name: "srv-sftp", State: utils.StateRunning.String()},
		{Name: "srv-r66", State: utils.StateError.String(), Reason: "bind: address already in use"},
	}
	clients = []internal.Service{{Name: "cli-sftp", State: utils.StateRunning.String()}}

	return core, servers, clients
}

type suite struct {
	db   *database.DB
	h    *Handlers
	now  time.Time
	node *stubNode

	monitorsTotal, monitorsFailing int
	monitorsErr                    error
	certEnd                        time.Time // when the certificate of the server ends: ten days from now, the real now

	failedRecent, failedOld, failedServer *model.Transfer
}

// stubNode is a node whose figures are chosen: started on the 9th at 06:12,
// twelve days and nine hours before the clock of the suite.
type stubNode struct {
	started  time.Time
	noStart  bool
	memUsed  uint64
	memTotal uint64
	noTotal  bool
	cpu      float64
	noCPU    bool
	cores    int
	in, out  float64
	connIn   int64
	connOut  int64
	dirs     []analytics.Dir
	usages   map[string]analytics.DiskUsage
}

func sampleNode() *stubNode {
	dirs := []analytics.Dir{{Name: analytics.DirIn, Path: "/gw/in"}, {Name: analytics.DirOut, Path: "/gw/out"}, {Name: analytics.DirTmp, Path: "/gw/tmp"}}

	return &stubNode{
		started: at(9, 6, 12), memUsed: 48 << 20, memTotal: 8 << 30, cpu: 0.03, cores: 4,
		in: 1200 << 10, out: 400 << 10, connIn: 3, connOut: 2, dirs: dirs,
		usages: map[string]analytics.DiskUsage{
			analytics.DirIn:  {Dir: dirs[0], UsedPercent: 81, Free: 7 << 30},
			analytics.DirOut: {Dir: dirs[1], UsedPercent: 41, Free: 22 << 30},
			analytics.DirTmp: {Dir: dirs[2], UsedPercent: 12, Free: 33 << 30},
		},
	}
}

func (n *stubNode) Version() string                { return "0.19.0-test" }
func (n *stubNode) StartTime() (time.Time, bool)   { return n.started, !n.noStart }
func (n *stubNode) MemoryUsed() uint64             { return n.memUsed }
func (n *stubNode) MemoryTotal() (uint64, bool)    { return n.memTotal, !n.noTotal }
func (n *stubNode) CPUShare() (float64, bool)      { return n.cpu, !n.noCPU }
func (n *stubNode) Cores() int                     { return n.cores }
func (n *stubNode) Throughput() (float64, float64) { return n.in, n.out }
func (n *stubNode) Connections() (int64, int64)    { return n.connIn, n.connOut }
func (n *stubNode) Dirs() []analytics.Dir          { return n.dirs }
func (n *stubNode) DiskUsageOf(name string) (analytics.DiskUsage, error) {
	usage, ok := n.usages[name]
	if !ok {
		return analytics.DiskUsage{}, analytics.ErrUnknownDir
	}

	return usage, nil
}

// newSuite builds the handlers on a fresh base, the clock stopped at 16:05 on
// the 21st: the window runs from 17:00 on the 20th to the hour under way.
func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{
		db: dbtest.TestDatabase(t), now: at(21, 16, 5), node: sampleNode(), monitorsTotal: 14, monitorsFailing: 1,
		certEnd: time.Now().Add(10 * 24 * time.Hour),
	}
	s.h = New(s.db, testhelpers.GetTestLogger(t), sampleServices, s.node,
		func() (int, int, error) { return s.monitorsTotal, s.monitorsFailing, s.monitorsErr })
	s.h.now = func() time.Time { return s.now }

	return s
}

// fixtures inserts a partner with its account and a client, a server with
// its account, two rules; live transfers of every state, three of them failed
// (one started half an hour ago, one three hours ago, a server one yesterday,
// before the window); history entries done in and out of the window, and one
// cancelled.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	bank := &model.RemoteAgent{Name: "banque-demo", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	require.NoError(t, s.db.Insert(bank).Run())

	prod := &model.RemoteAccount{RemoteAgentID: bank.ID, Login: "prod"}
	require.NoError(t, s.db.Insert(prod).Run())

	client := &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	require.NoError(t, s.db.Insert(client).Run())

	server := &model.LocalAgent{Name: "srv-sftp", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2222)}
	require.NoError(t, s.db.Insert(server).Run())

	south := &model.LocalAccount{LocalAgentID: server.ID, Login: "filiale-sud"}
	require.NoError(t, s.db.Insert(south).Run())

	// The certificates of the node (#836): the server one ends ten days from
	// the real now (a certificate must be valid when it is written), the
	// authority's one in years.
	https := &model.LocalAgent{Name: "srv-https", Protocol: "https", Address: types.Addr("127.0.0.1", 8443)}
	require.NoError(t, s.db.Insert(https).Run())

	cert, key := certPEM(t, "srv-https", s.certEnd)
	require.NoError(t, s.db.Insert(&model.Credential{
		LocalAgentID: https.NullableID(), Name: "cert-https", Type: auth.TLSCertificate, Value: cert, Value2: key,
	}).Run())

	caCert, _ := certPEM(t, "ca-interne", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, s.db.Insert(&model.Authority{Name: "ca-interne", Type: auth.AuthorityTLS, PublicIdentity: caCert}).Run())

	send := &model.Rule{Name: "releves", IsSend: true, Path: "/releves"}
	receive := &model.Rule{Name: "commandes", IsSend: false, Path: "/commandes"}
	require.NoError(t, s.db.Insert(send).Run())
	require.NoError(t, s.db.Insert(receive).Run())

	clientTransfer := func(file string, status types.TransferStatus, start time.Time) *model.Transfer {
		return &model.Transfer{
			RuleID: send.ID, ClientID: client.NullableID(), RemoteAccountID: prod.NullableID(),
			SrcFilename: file, DestFilename: file, Status: status, Start: start, Filesize: model.UnknownSize,
		}
	}

	s.failedRecent = clientTransfer("releve-20102026.pdf", types.StatusError, at(21, 15, 35))
	s.failedRecent.ErrCode, s.failedRecent.ErrDetails = types.TeConnection, "dial tcp 127.0.0.1:10022: connection refused"
	s.failedRecent.RemainingTries = 2

	s.failedOld = clientTransfer("paie-septembre.xml", types.StatusError, at(21, 13, 5))
	s.failedOld.ErrCode = types.TeBadAuthentication

	s.failedServer = &model.Transfer{
		RuleID: receive.ID, LocalAccountID: south.NullableID(), DestFilename: "commandes-0457.edi",
		Status: types.StatusError, Start: at(20, 10, 0), ErrCode: types.TeExternalOperation,
		ErrDetails: "post-task exit 2",
	}

	for _, trans := range []*model.Transfer{
		clientTransfer("paie-octobre.xml", types.StatusRunning, at(21, 15, 40)),
		clientTransfer("rapport-trimestriel.pdf", types.StatusRunning, at(21, 12, 50)), // stuck: over three hours
		clientTransfer("rapport-mensuel.pdf", types.StatusPlanned, at(21, 18, 0)),
		clientTransfer("rapport-annuel.pdf", types.StatusPlanned, at(21, 19, 0)),
		clientTransfer("rapport-hebdo.pdf", types.StatusPlanned, at(21, 14, 0)), // overdue: two hours ago
		clientTransfer("archive.zip", types.StatusPaused, at(21, 9, 0)),
		clientTransfer("stocks.csv", types.StatusInterrupted, at(21, 11, 30)),
		{
			RuleID: receive.ID, LocalAccountID: south.NullableID(), DestFilename: "inventaire.csv",
			Status: types.StatusAvailable, Start: at(21, 23, 0),
		},
		s.failedRecent, s.failedOld, s.failedServer,
	} {
		require.NoError(t, s.db.Insert(trans).Run())
	}

	done := func(id int64, file string, start time.Time, size int64) *model.HistoryEntry {
		return &model.HistoryEntry{
			Identifier: model.ID(id), RemoteTransferID: "r-" + file, IsSend: true, Rule: "releves", Account: "prod",
			Agent: "banque-demo", Client: "cli-sftp", Protocol: "sftp", SrcFilename: file, DestFilename: file,
			Filesize: size, Progress: size, Start: start, Stop: start.Add(time.Minute), Status: types.StatusDone,
		}
	}

	cancelled := done(1005, "virements.xml", at(21, 13, 10), model.UnknownSize)
	cancelled.Status = types.StatusCancelled

	for _, hist := range []*model.HistoryEntry{
		done(1001, "releve-1.pdf", at(21, 15, 30), 1_000_000),
		done(1002, "releve-2.pdf", at(21, 14, 40), 2_000_000),
		done(1003, "releve-3.pdf", at(20, 17, 30), 500_000),  // the first hour of the window
		done(1004, "releve-4.pdf", at(20, 15, 0), 4_000_000), // before the window
		cancelled,
	} {
		require.NoError(t, s.db.Insert(hist).Run())
	}
}

// serve runs the page handler as a user with the given rights.
func (s *suite) serve(t *testing.T, perms model.PermsMask, lang, target string) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	rec := httptest.NewRecorder()
	s.h.Page(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, target, nil))

	return rec
}

// indexes asserts that the markers appear in the body in the given order.
func indexes(t *testing.T, body string, markers ...string) {
	t.Helper()

	last := -1

	for _, marker := range markers {
		pos := strings.Index(body, marker)
		require.NotEqualf(t, -1, pos, "%q not found in the page", marker)
		assert.Greaterf(t, pos, last, "%q is out of order", marker)
		last = pos
	}
}

// part returns the markup of an element of the body, by its id: the element
// and what it contains, its nested elements of the same name counted.
func part(t *testing.T, body, id string) string {
	t.Helper()

	at := strings.Index(body, ` id="`+id+`"`)
	require.NotEqualf(t, -1, at, "part %q not found", id)

	open := strings.LastIndex(body[:at], "<")
	name := body[open+1 : open+1+strings.IndexAny(body[open+1:], " >")]
	depth := 0

	for i := open; i < len(body); i++ {
		switch {
		case strings.HasPrefix(body[i:], "<"+name+" ") || strings.HasPrefix(body[i:], "<"+name+">"):
			depth++
		case strings.HasPrefix(body[i:], "</"+name+">"):
			depth--
			if depth == 0 {
				return body[open : i+len(name)+3]
			}
		}
	}

	t.Fatalf("part %q is not closed", id)

	return ""
}

func TestWindowStart(t *testing.T) {
	t.Parallel()

	assert.Equal(t, at(20, 17, 0), windowStart(at(21, 16, 5)), "23 whole hours before the hour under way")
	assert.Equal(t, at(20, 1, 0), windowStart(at(21, 0, 59)))
}

func TestPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	read, write := model.PermTransfersRead, model.PermTransfersRead|model.PermTransfersWrite

	t.Run("Given a user who may act on the transfers, in French", func(t *testing.T) {
		rec := s.serve(t, write, "fr", "/webui/dashboard")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, "<title>Tableau de bord · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<h1 class="w-page-header__title">Tableau de bord</h1>`)
		assert.Contains(t, body, `<li><a href="/webui/home">Accueil</a></li><li><span>Supervision</span></li><li><span aria-current="page">Tableau de bord</span></li>`)
		assert.Contains(t, body, `id="tdb-live" role="status" aria-live="polite"`)
		assert.Contains(t, body, `>Actualisé à 16:05:00</span>`)
		assert.Contains(t, body, `id="tdb-refresh" href="/webui/dashboard">Actualiser</a>`)
		assert.Contains(t, body, `data-refresh-seconds="30" data-refreshed-at="16:05:00"`)
		// The grouped resumption (#818): the only coloured action, saying how many
		// will resume (the server transfer apart), the failed count it was shown.
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"), "one coloured action: the grouped resumption")
		indexes(t, body, `id="tdb-refresh"`, `<div id="tdb-resume" class="tdb-resume"><form method="post" action="/webui/transfers/resume?from=dashboard">`,
			`<input type="hidden" name="state" value="error">`, `<input type="hidden" name="all" value="1">`,
			`<input type="hidden" name="expected" value="3">`, `Reprendre les 2 transferts en erreur</button>`, `id="tdb-body"`)
		assert.NotContains(t, body, "¿", "no missing key")

		// The health of the node: the strip of the kit, a cell per figure (#834).
		health := part(t, body, "tdb-health")
		assert.Contains(t, body, `<h2 class="w-section-title">Santé du nœud gw-test</h2>`)
		assert.Equal(t, 8, strings.Count(health, `class="w-health__item `), "eight cells")
		indexes(t, health,
			`<a class="w-health__item w-health__item--ok" href="/webui/about" title="Démarrée le 09/10/2026 à 06:12:00 ; version 0.19.0-test"><span class="w-health__label">Disponibilité</span> <span class="w-health__value">12 j 09 h</span> <span class="w-status w-status--ok">v0.19.0-test, depuis le 09/10/2026</span></a>`,
			`<a class="w-health__item w-health__item--ko" href="/webui/services" title="Services système 2 / 2 · Serveurs locaux 1 / 2 · Clients locaux 1 / 1"><span class="w-health__label">Services</span> <span class="w-health__value">4<small>/ 5</small></span> <span class="w-status w-status--ko">1 en erreur</span></a>`,
			`<a class="w-health__item w-health__item--ok" href="/webui/services?group=core" title="Base memory ; dernière lecture des chiffres en `,
			`<span class="w-health__label">Base de données</span>`, `<small>ms</small></span> <span class="w-status w-status--ok">Répond</span></a>`,
			`<div class="w-health__item w-health__item--ok" title="Part du processeur et mémoire prises par la passerelle"><span class="w-health__label">Processeur · mémoire</span> <span class="w-health__value">3<small>% · 48 Mo</small></span> <span class="w-status w-status--ok">Sur 4 cœurs et 8,0 Go</span></div>`,
			`<div class="w-health__item w-health__item--warn" title="in 81 % (7,0 Go libres) · out 41 % (22 Go libres) · tmp 12 % (33 Go libres)"><span class="w-health__label">Espace disque</span> <span class="w-health__value">81<small>%</small></span> <span class="w-status w-status--warn">in : 7,0 Go libres</span></div>`,
			`<div class="w-health__item w-health__item--ok" title="Moyenne des 30 dernières secondes ; 3 connexions entrantes, 2 sortantes"><span class="w-health__label">Réseau</span> <span class="w-health__value">1,2 Mo/s<small>reçus</small></span> <span class="w-status w-status--ok">400 Ko/s émis · 5 connexions</span></div>`,
			`<a class="w-health__item w-health__item--ko" href="/webui/watchers"><span class="w-health__label">Surveillances</span> <span class="w-health__value">13<small>/ 14</small></span> <span class="w-status w-status--ko">1 en défaut</span></a>`)

		assert.Contains(t, body, `<h2 class="w-section-title">Transferts</h2>`)
	})

	t.Run("Given a node whose figures cannot all be read", func(t *testing.T) {
		s.node.noStart, s.node.noCPU, s.node.noTotal = true, true, true
		s.node.dirs, s.monitorsErr = nil, errors.New("registry down")
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		health := part(t, body, "tdb-health")

		assert.Contains(t, health, `<span class="w-health__label">Disponibilité</span> <span class="w-health__value">Non mesuré</span> <span class="w-status w-status--off">v0.19.0-test</span>`)
		assert.Contains(t, health, `<span class="w-health__label">Processeur · mémoire</span> <span class="w-health__value">48 Mo</span> <span class="w-status w-status--off">Processeur non mesuré</span>`)
		assert.Contains(t, health, `<span class="w-health__label">Espace disque</span> <span class="w-health__value">Non mesuré</span> <span class="w-status w-status--off">Aucun dossier mesuré</span>`)
		assert.Contains(t, health, `<span class="w-health__label">Surveillances</span> <span class="w-health__value">Non mesuré</span> <span class="w-status w-status--off">État inconnu</span>`)

		s.node, s.monitorsErr = sampleNode(), nil
		s.h.info = s.node
	})

	t.Run("Given the certificates of the node", func(t *testing.T) {
		// Judged at the real now, the clock the certificate was written with:
		// the server certificate ends in ten days, the authority's in years.
		_, date := transfers.TimeAndDate(func(key string, _ ...any) string { return "02/01/2006" }, s.certEnd)
		s.now, s.h.last = time.Now(), reading{}
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		health := part(t, body, "tdb-health")

		assert.Contains(t, health, `<div class="w-health__item w-health__item--warn" title="serveur srv-https : `+date+`"><span class="w-health__label">Certificats</span> <span class="w-health__value">1<small>/ 2</small></span> <span class="w-status w-status--warn">Expire sous 30 jours</span></div>`)

		// Twenty days on, it has expired.
		s.now, s.h.last = time.Now().Add(20*24*time.Hour), reading{}
		body = s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		health = part(t, body, "tdb-health")

		assert.Contains(t, health, `<div class="w-health__item w-health__item--ko" title="serveur srv-https : `+date+`"><span class="w-health__label">Certificats</span> <span class="w-health__value">1<small>/ 2</small></span> <span class="w-status w-status--ko">1 expiré</span></div>`)

		s.now, s.h.last = at(21, 16, 5), reading{}
	})

	t.Run("Given the thresholds of the strip", func(t *testing.T) {
		s.node.cpu, s.node.usages[analytics.DirTmp] = 3.3, analytics.DiskUsage{Dir: s.node.dirs[2], UsedPercent: 93, Free: 2 << 30}
		s.monitorsFailing = 0
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		health := part(t, body, "tdb-health")

		assert.Contains(t, health, `<span class="w-health__label">Processeur · mémoire</span> <span class="w-health__value">330<small>% · 48 Mo</small></span> <span class="w-status w-status--warn">`, "3.3 cores of 4: warning")
		assert.Contains(t, health, `w-health__item--ko" title="in 81 % (7,0 Go libres) · out 41 % (22 Go libres) · tmp 93 % (2,0 Go libres)"><span class="w-health__label">Espace disque</span> <span class="w-health__value">93<small>%</small></span> <span class="w-status w-status--ko">tmp : 2,0 Go libres</span>`, "the fullest directory, at 93 %: failure")
		assert.Contains(t, health, `<span class="w-health__label">Surveillances</span> <span class="w-health__value">14<small>/ 14</small></span> <span class="w-status w-status--ok">Toutes actives</span>`)

		s.node = sampleNode()
		s.h.info = s.node
		s.monitorsFailing = 1
	})

	t.Run("Given the tiles of the transfers", func(t *testing.T) {
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		tiles := part(t, body, "tdb-transfers")

		running := regexp.MustCompile(`(?s)<a class="w-card w-tile w-tile--info" href="/webui/transfers\?state=running">.*?</a>`).FindString(tiles)
		require.NotEmpty(t, running)
		assert.Contains(t, running, `<span class="w-tile__label">Transferts en cours</span>`)
		assert.Contains(t, running, `<span aria-hidden="true">2</span><span class="visually-hidden">2 transferts en cours</span>`)
		indexes(t, running, ">3 planifiés<", ">1 en pause<", ">1 interrompu<", ">1 attendu<",
			`<span class="w-status w-status--warn">1 en cours depuis plus d&#39;une heure</span>`,
			`<span class="w-status w-status--warn">1 planifié en retard</span>`)

		failedTile := regexp.MustCompile(`(?s)<div class="w-card w-tile w-tile--ko"><a class="w-tile__main" href="/webui/transfers\?state=error">.*?</div>`).FindString(tiles)
		require.NotEmpty(t, failedTile)
		assert.Contains(t, failedTile, `<span class="w-muted"><a class="w-link" href="/webui/transfers?role=client&amp;state=error">2 côté client</a> · `+
			`<a class="w-link" href="/webui/transfers?role=server&amp;state=error">1 côté serveur</a></span>`,
			"the roles of the failed transfers (#835), each leading to the monitoring filtered by its role (#837)")

		doneTile := regexp.MustCompile(`(?s)<a class="w-card w-tile w-tile--ok" href="/webui/transfers\?from=[^"]+state=done">.*?</a>`).FindString(tiles)
		require.NotEmpty(t, doneTile)
		assert.Contains(t, doneTile, `<span class="w-status w-status--ok">Dernier terminé à 15:30</span>`, "the last transfer done (#835)")

		failed := regexp.MustCompile(`(?s)<div class="w-card w-tile w-tile--ko"><a class="w-tile__main" href="/webui/transfers\?state=error">.*?</div>`).FindString(tiles)
		require.NotEmpty(t, failed)
		assert.Contains(t, failed, `<span aria-hidden="true">3</span><span class="visually-hidden">3 transferts en erreur</span>`)
		assert.Contains(t, failed, `<span class="w-status w-status--ko">1 depuis une heure</span>`)

		done := regexp.MustCompile(`(?s)<a class="w-card w-tile w-tile--ok" href="/webui/transfers\?from=2026-10-20T17%3A00&amp;state=done">.*?</a>`).FindString(tiles)
		require.NotEmpty(t, done, "the list of the done transfers is bounded by the window")
		assert.Contains(t, done, `<span class="w-tile__label">Terminés sur 24 h</span>`)
		assert.Contains(t, done, `<span aria-hidden="true">3</span>`, "the transfer of yesterday 15:00 is out, the one of 17:30 is in")
		assert.Contains(t, done, `<span class="w-status w-status--off">1 annulé</span>`)
		assert.Contains(t, done, `<span class="w-muted">3,3 Mo transférés</span>`)
	})

	t.Run("Given the activity of the window", func(t *testing.T) {
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()

		activity := part(t, body, "tdb-activity")
		assert.Contains(t, activity, `<svg class="w-chart w-chart--wide" viewBox="0 0 600 240" role="img"`, "grouped by day")
		assert.Contains(t, activity, `<svg class="w-chart w-chart--narrow"`, "24 slots: a narrow rendering too")
		indexes(t, activity, `>Hier · 20/10</text>`, `>Aujourd&#39;hui · 21/10</text>`)
		assert.Contains(t, activity, `<line class="group"`, "a line at midnight")
		assert.Contains(t, activity, `<title id="tdb-activity-chart-t">Transferts par heure sur les 24 dernières heures : terminés et encore en erreur</title>`)
		assert.Contains(t, activity, "15:00 · 1 terminés", "the transfer done at 15:30 today")
		assert.Contains(t, activity, "15:00 · 1 en erreur", "the transfer failed at 15:35 today, hatched")
		assert.Contains(t, activity, "13:00 · 1 en erreur", "the transfer failed at 13:05")
		assert.Contains(t, activity, "17:00 · 1 terminés", "the transfer done yesterday at 17:30, first hour of the window")
		assert.NotContains(t, activity, "10:00 · 1 en erreur", "the server transfer failed yesterday morning is out of the window")
		assert.Contains(t, activity, `>Terminés</span>`)
		assert.Contains(t, activity, `>Encore en erreur (hachuré, chiffre au-dessus)</span>`)

		assert.Contains(t, body, `<details class="w-fold tdb-values" data-key="tdb-values">`)
		assert.NotContains(t, body, `<details class="w-fold tdb-values" data-key="tdb-values" open`, "closed by default")
		assert.Contains(t, body, `<h3 class="w-fold__title">Valeurs par heure</h3><span class="w-fold__meta" id="tdb-values-meta" data-tdb-part>3 terminés · 2 en erreur</span>`)

		values := part(t, body, "tdb-values")
		assert.Equal(t, 24, strings.Count(values, "<tr><td"), "one row per hour")
		assert.Equal(t, 2, strings.Count(values, "<tbody>"), "a group per day")
		indexes(t, values,
			`<tr class="tdb-day"><th scope="rowgroup" colspan="3">Aujourd&#39;hui · 21/10<span class="tdb-day__meta">2 terminés · 2 en erreur</span></th></tr>`,
			`<td class="w-mono">16:00</td><td class="w-num">0</td><td class="w-num">0</td>`,
			`<td class="w-mono">15:00</td><td class="w-num">1</td><td class="w-num">1</td>`,
			`<td class="w-mono">13:00</td><td class="w-num">0</td><td class="w-num">1</td>`,
			`<td class="w-mono">00:00</td>`,
			`<tr class="tdb-day"><th scope="rowgroup" colspan="3">Hier · 20/10<span class="tdb-day__meta">1 terminé · 0 en erreur</span></th></tr>`,
			`<td class="w-mono">23:00</td>`,
			`<td class="w-mono">17:00</td><td class="w-num">1</td><td class="w-num">0</td>`)
	})

	t.Run("Given a window that starts at midnight", func(t *testing.T) {
		s.now = at(21, 23, 30)
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()

		activity := part(t, body, "tdb-activity")
		assert.Contains(t, activity, `>Aujourd&#39;hui · 21/10</text>`, "one group, today")
		assert.NotContains(t, activity, "Hier")
		assert.NotContains(t, activity, `<line class="group"`, "no boundary with one group")

		values := part(t, body, "tdb-values")
		assert.Equal(t, 1, strings.Count(values, "<tbody>"))
		indexes(t, values, `<td class="w-mono">23:00</td>`, `<td class="w-mono">00:00</td>`)

		s.now = at(21, 16, 5)
		s.h.last = reading{} // the figures of 23:30 are not those of the clock set back
	})

	t.Run("Given the latest errors", func(t *testing.T) {
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		errors := part(t, body, "tdb-errors")

		assert.Equal(t, 3, strings.Count(errors, "<tr><td"), "the three failed transfers")
		indexes(t, errors, "releve-20102026.pdf", "paie-septembre.xml", "commandes-0457.edi")
		assert.Contains(t, errors, `<span class="w-mono">15:35:00</span>`)
		assert.Contains(t, errors, `<div class="w-cell-stack"><span class="w-mono">10:00:00</span><span class="w-cell-sub">20/10/2026</span></div>`,
			"the date of a transfer not started today")
		assert.Contains(t, errors, `<a class="w-link" href="/webui/transfers/`+itoa(s.failedRecent.ID)+`">releve-20102026.pdf</a></span><span class="w-cell-sub">releves · banque-demo</span>`)
		assert.Contains(t, errors, `<span class="w-cell-sub">commandes · srv-sftp</span>`, "a server transfer names its server")
		assert.Contains(t, errors, `<span class="w-cell-sub tdb-detail" title="dial tcp 127.0.0.1:10022: connection refused">dial tcp 127.0.0.1:10022: connection refused</span>`)

		recent := regexp.MustCompile(`(?s)<div class="w-actions" id="tdb-act-` + itoa(s.failedRecent.ID) + `" popover>.*?</div>`).FindString(errors)
		require.NotEmpty(t, recent)
		indexes(t, recent, `href="/webui/transfers/`+itoa(s.failedRecent.ID)+`"`,
			`<form method="post" action="/webui/transfers/`+itoa(s.failedRecent.ID)+`/resume?from=dashboard">`,
			`href="/webui/transfers/`+itoa(s.failedRecent.ID)+`/retry?from=dashboard"`,
			`<hr>`, `href="/webui/transfers/`+itoa(s.failedRecent.ID)+`/cancel?from=dashboard"`)

		server := regexp.MustCompile(`(?s)<div class="w-actions" id="tdb-act-` + itoa(s.failedServer.ID) + `" popover>.*?</div>`).FindString(errors)
		require.NotEmpty(t, server)
		assert.NotContains(t, server, "/resume", "a server transfer resumes on the initiative of the partner")
		assert.NotContains(t, server, "/retry")
		assert.Contains(t, server, "/cancel?from=dashboard")

		assert.Contains(t, errors, `<p class="tdb-foot"><a class="w-link" href="/webui/transfers?state=error">Voir les 3 transferts en erreur</a></p>`)
	})

	t.Run("Given a user who reads the transfers only", func(t *testing.T) {
		body := s.serve(t, read, "en", "/webui/dashboard").Body.String()
		errors := part(t, body, "tdb-errors")

		assert.NotContains(t, errors, "popover", "no menu without the right to write")
		assert.NotContains(t, body, "/webui/transfers/resume", "no grouped resumption either")
		assert.Contains(t, body, `<div id="tdb-resume" class="tdb-resume"></div>`)
		assert.Equal(t, 3, strings.Count(errors, `aria-label="View the transfer"`), "the sheet alone")
		assert.Contains(t, body, "Health of node gw-test")
		assert.Contains(t, body, `<span class="w-health__label">Uptime</span> <span class="w-health__value">12 d 09 h</span>`)
		assert.Contains(t, body, `<div class="w-main__inner w-main__inner--wide">`, "the whole width (#833)")
		assert.Contains(t, body, `<span class="w-tile__label">Done in 24 h</span>`)
		assert.Contains(t, body, `>3 done · 2 failed</span>`)
		assert.Contains(t, body, `>View the 3 failed transfers</a>`)
	})

	t.Run("Given a user without the right to read the transfers", func(t *testing.T) {
		s.h.last = reading{} // nothing read yet: the database cell says so
		body := s.serve(t, 0, "fr", "/webui/dashboard").Body.String()

		assert.Contains(t, body, `id="tdb-health"`)
		assert.Contains(t, body, `<span class="w-health__label">Base de données</span> <span class="w-health__value">Non mesuré</span> <span class="w-status w-status--off">Pas encore lue</span>`,
			"the database is not read for a user who does not read the transfers")
		assert.Contains(t, body, `<span class="w-health__label">Certificats</span> <span class="w-health__value">Non mesuré</span> <span class="w-status w-status--off">Non lus</span>`,
			"nor the certificates")
		assert.NotContains(t, body, `id="tdb-transfers"`)
		assert.NotContains(t, body, "tdb-errors")
		assert.NotContains(t, part(t, body, "tdb-notice"), "w-banner", "no message: the sidebar does not offer the transfers either")
	})

	t.Run("Given certificates the base cannot read", func(t *testing.T) {
		// A credential whose key is not ciphered as the base expects: the
		// certificates are unread, the other figures stand.
		require.NoError(t, s.db.Exec("UPDATE credentials SET value2=? WHERE name=?", "not-a-ciphered-key", "cert-https"))
		s.h.last = reading{}
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()

		assert.Contains(t, body, `<span class="w-health__label">Certificats</span> <span class="w-health__value">Non mesuré</span> <span class="w-status w-status--off">Non lus</span>`)
		assert.Contains(t, body, `id="tdb-transfers"`, "the figures of the transfers stand")
		assert.NotContains(t, body, "Injoignable")

		cert, key := certPEM(t, "srv-https", s.certEnd)
		require.NoError(t, s.db.Exec("DELETE FROM credentials WHERE name=?", "cert-https"))
		require.NoError(t, s.db.Insert(&model.Credential{
			LocalAgentID: utils.NewNullInt64(localAgentID(t, s.db, "srv-https")), Name: "cert-https",
			Type: auth.TLSCertificate, Value: cert, Value2: key,
		}).Run())
		s.h.last = reading{}
	})

	t.Run("Given a request for the body fragment", func(t *testing.T) {
		rec := s.serve(t, write, "fr", "/webui/dashboard?fragment=body")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.NotContains(t, body, "<html")
		assert.True(t, strings.HasPrefix(body, `<div id="tdb-body"`), "the body first")
		assert.Contains(t, body, `id="tdb-errors"`)
		assert.Contains(t, body, `<div id="tdb-resume" class="tdb-resume"><form`, "then the resumption button, which follows the figures")
	})

	t.Run("Given a failed child of a synchronous transfer", func(t *testing.T) {
		other := newSuite(t)
		other.fixtures(t)

		child := &model.Transfer{
			RuleID: other.failedRecent.RuleID, ClientID: other.failedRecent.ClientID,
			RemoteAccountID: other.failedRecent.RemoteAccountID, SrcFilename: "lot-2.pdf", DestFilename: "lot-2.pdf",
			Status: types.StatusError, Start: at(21, 15, 50), Filesize: model.UnknownSize,
			TransferInfo: map[string]any{model.SyncTransferID: other.failedRecent.ID, model.SyncTransferRank: int64(1)},
		}
		require.NoError(t, other.db.Insert(child).Run())

		body := other.serve(t, write, "en", "/webui/dashboard").Body.String()
		indexes(t, body, `<input type="hidden" name="expected" value="4">`, `Resume the 2 failed transfers</button>`)
	})
}

func TestEmptyPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	body := s.serve(t, model.PermTransfersRead|model.PermTransfersWrite, "fr", "/webui/dashboard").Body.String()

	tiles := part(t, body, "tdb-transfers")
	assert.Contains(t, tiles, `w-tile--ok" href="/webui/transfers?state=error"`, "no failure: the tile is fine")
	assert.Contains(t, tiles, `<span class="w-status w-status--ok">Aucun transfert en erreur</span>`)
	assert.Equal(t, 3, strings.Count(tiles, `<span aria-hidden="true">0</span>`))

	activity := part(t, body, "tdb-activity")
	assert.NotContains(t, activity, "w-chart", "no chart without any transfer")
	assert.Contains(t, activity, `<p class="w-empty__title">Aucun transfert sur les 24 dernières heures</p>`)
	assert.NotContains(t, body, `data-key="tdb-values"`, "no values either")

	errors := part(t, body, "tdb-errors")
	assert.Contains(t, errors, `<p class="w-empty__title">Aucun transfert en erreur</p>`)
	assert.NotContains(t, errors, "<table")
	assert.NotContains(t, body, "w-btn--primary", "nothing to resume: no button")
}

func TestFiguresCache(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	figs, last := s.h.figures(s.now, false)
	require.NotNil(t, figs)
	assert.EqualValues(t, 3, figs.Live[statusKey(types.StatusError)])
	assert.Equal(t, s.now, last.at)
	assert.True(t, last.failing.IsZero())

	// One failure more: the figures are those of the period until it ends.
	require.NoError(t, s.db.Insert(&model.Transfer{
		RuleID: s.failedOld.RuleID, ClientID: s.failedOld.ClientID, RemoteAccountID: s.failedOld.RemoteAccountID,
		SrcFilename: "late.csv", DestFilename: "late.csv", Status: types.StatusError, Start: at(21, 16, 0), Filesize: model.UnknownSize,
	}).Run())

	figs, _ = s.h.figures(s.now.Add(20*time.Second), false)
	assert.EqualValues(t, 3, figs.Live[statusKey(types.StatusError)], "read once per period")

	figs, last = s.h.figures(s.now.Add(31*time.Second), false)
	assert.EqualValues(t, 4, figs.Live[statusKey(types.StatusError)], "read again once the period is over")
	assert.Equal(t, s.now.Add(31*time.Second), last.at)

	// An action came back with its result: read again at once (#818).
	require.NoError(t, s.db.Insert(&model.Transfer{
		RuleID: s.failedOld.RuleID, ClientID: s.failedOld.ClientID, RemoteAccountID: s.failedOld.RemoteAccountID,
		SrcFilename: "later.csv", DestFilename: "later.csv", Status: types.StatusError, Start: at(21, 16, 1), Filesize: model.UnknownSize,
	}).Run())

	figs, last = s.h.figures(s.now.Add(40*time.Second), true)
	assert.EqualValues(t, 5, figs.Live[statusKey(types.StatusError)], "read again after an action, within the period")
	assert.Equal(t, s.now.Add(40*time.Second), last.at)

	body := s.serve(t, model.PermTransfersRead|model.PermTransfersWrite, "fr", "/webui/dashboard?done=set_resumed&name=3").Body.String()
	assert.Contains(t, body, "3 transferts repris", "the result of the action")
}

func TestDegradedPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	write := model.PermTransfersRead | model.PermTransfersWrite

	first := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
	assert.NotContains(t, part(t, first, "tdb-notice"), "w-banner", "the base answers: no message")

	// The base stops answering: the page keeps its figures, dated, and says
	// since when the base fails.
	s.h.read = func(time.Time) (*figures, error) { return nil, errors.New("database is closed") }

	s.now = at(21, 16, 7)
	body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()

	notice := part(t, body, "tdb-notice")
	assert.Contains(t, notice, `<div class="w-banner w-banner--ko" role="alert">`)
	assert.Contains(t, notice, `<strong>La base de données ne répond plus depuis 16:07:00.</strong>Les chiffres affichés datent de 16:05:00.`)
	assert.Contains(t, notice, `href="/webui/services">`)
	assert.Contains(t, body, `<span aria-hidden="true">3</span><span class="visually-hidden">3 transferts en erreur</span>`,
		"the last figures read stay")
	assert.Contains(t, body, ">Actualisé à 16:07:00</span>", "the page says when it was rendered")

	// Later, still failing: the first failure is the one named.
	s.now = at(21, 16, 9)
	body = s.serve(t, write, "fr", "/webui/dashboard").Body.String()
	assert.Contains(t, body, "La base de données ne répond plus depuis 16:07:00.")

	// A base that never answered: the message alone, without figures.
	fresh := newSuite(t)
	fresh.h.read = func(time.Time) (*figures, error) { return nil, errors.New("database is closed") }
	body = fresh.serve(t, write, "fr", "/webui/dashboard").Body.String()
	assert.Contains(t, body, "Aucun chiffre n&#39;a encore pu être lu.")
	assert.Contains(t, body, `id="tdb-health"`, "the health of the node, read in memory, stays")
	assert.Contains(t, body, `<span class="w-health__label">Base de données</span> <span class="w-health__value">Injoignable</span> <span class="w-status w-status--ko">Depuis 16:05:00</span>`)
	assert.NotContains(t, body, `id="tdb-transfers"`)
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

// localAgentID is the id of the server of that name.
func localAgentID(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()

	var server model.LocalAgent
	require.NoError(t, db.Get(&server, "name=?", name).Run())

	return server.ID
}
