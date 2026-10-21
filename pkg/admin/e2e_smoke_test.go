//go:build e2e

package admin_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/mxschmitt/playwright-go"
	"github.com/rclone/rclone/fs/object"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/fs/backends" // the description of the storages: the cloud form lists their options
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
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
		{path: "/webui/account", outside: true},
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
		// The users (#718): the list, the sheet of the smoke user (user 2: the
		// cards of their rights), the pages that add, edit and delete the
		// viewer (user 3), the refused deletion of the smoke user's own
		// account.
		{path: "/webui/users", tables: 1},
		{path: "/webui/users/2"},
		{path: "/webui/users/new"},
		{path: "/webui/users/3/edit"},
		{path: "/webui/users/3/delete"},
		{path: "/webui/users/2/delete"},
		// The cloud instances (#719): the list, the sheet of the S3 instance
		// (named by a rule), the pages that add, edit and delete it, the
		// sheet of the OneDrive instance (its token never shown).
		{path: "/webui/clouds", tables: 1},
		{path: "/webui/clouds/1"},
		{path: "/webui/clouds/new", tables: 5}, // the options the storage library accepts, one list per storage
		{path: "/webui/clouds/1/edit", tables: 5},
		{path: "/webui/clouds/1/delete"},
		{path: "/webui/clouds/2"},
		// The authentication authorities (#722): the list, the sheet of the
		// TLS authority (with the partner certificate it signs), the sheet of
		// the SSH authority, the pages that add, edit and delete one.
		{path: "/webui/authorities", tables: 1},
		{path: "/webui/authorities/1"},
		{path: "/webui/authorities/2"},
		{path: "/webui/authorities/new"},
		{path: "/webui/authorities/1/edit"},
		{path: "/webui/authorities/2/delete"},
		// SNMP (#723): the page of the monitors and of the agent, the sheets
		// of the SNMPv2c and SNMPv3 monitors, the pages that add, edit and
		// delete a monitor, the pages that configure and disable the agent.
		{path: "/webui/snmp", tables: 1},
		{path: "/webui/snmp/monitors/1"},
		{path: "/webui/snmp/monitors/2"},
		{path: "/webui/snmp/monitors/new"},
		{path: "/webui/snmp/monitors/2/edit"},
		{path: "/webui/snmp/monitors/1/delete"},
		{path: "/webui/snmp/agent/edit"},
		{path: "/webui/snmp/agent/delete"},
		// The configuration overrides (#730): the gateway runs as a node of
		// a cluster with one rule (the host of the servers and partners of
		// the smoke test); the list, the pages that add, edit and delete a
		// rule.
		{path: "/webui/overrides", tables: 1},
		{path: "/webui/overrides/new"},
		{path: "/webui/overrides/edit?target=127.0.0.1"},
		{path: "/webui/overrides/delete?target=127.0.0.1"},
		// The cryptographic keys (#735): the list, the sheet of the AES key
		// (the rule whose tasks use it: a table), the sheet of the PGP public
		// key, the pages that add and edit one, the deletion refused for the
		// AES key (the rules to change first) and confirmed for the PGP one.
		{path: "/webui/keys", tables: 1},
		{path: "/webui/keys/1"},
		{path: "/webui/keys/2"},
		{path: "/webui/keys/new"},
		{path: "/webui/keys/1/edit"},
		{path: "/webui/keys/1/delete"},
		{path: "/webui/keys/2/delete"},
		// The SMTP credentials (#741): the list, the sheet of the credential
		// a rule uses (the rule: a table), the sheet of the one no rule uses,
		// the pages that add and edit one, the deletion refused for the first
		// (the rules to change first) and confirmed for the second.
		{path: "/webui/smtp", tables: 1},
		{path: "/webui/smtp/1"},
		{path: "/webui/smtp/2"},
		{path: "/webui/smtp/new"},
		{path: "/webui/smtp/1/edit"},
		{path: "/webui/smtp/1/delete"},
		{path: "/webui/smtp/2/delete"},
		// The email templates (#744): the list, the sheet of the template a
		// rule uses (the rule: a table) with its preview, the sheet of the one
		// no rule uses, the pages that add and edit one (the rich text field
		// and the variables of the kit), the deletion refused for the first
		// (the rules to change first) and confirmed for the second.
		{path: "/webui/email-templates", tables: 1},
		{path: "/webui/email-templates/1", tables: 1},
		{path: "/webui/email-templates/2"},
		{path: "/webui/email-templates/new"},
		{path: "/webui/email-templates/1/edit"},
		{path: "/webui/email-templates/1/delete", tables: 1},
		{path: "/webui/email-templates/2/delete"},
	}
}

// insertE2EEmailTemplates inserts an HTML template with attachments, used by
// no rule, next to the text one insertE2ESMTP inserts for its rule. The
// screens are visited by identifier.
func insertE2EEmailTemplates(t *testing.T, db *database.DB) {
	t.Helper()

	tpl := &model.EmailTemplate{
		Name: "rapport-quotidien", Subject: "Rapport du #DATE#", MIMEType: "text/html",
		Body:        "<p>Le fichier <b>#TRUEFILENAME#</b> de la règle #RULE# est arrivé.</p>",
		Attachments: []string{"#TRUEFULLPATH#"},
	}
	require.NoError(t, db.Insert(tpl).Run())
	require.EqualValues(t, 2, tpl.ID, "the screens are visited by identifier")
}

// insertE2ESMTP inserts an SMTP credential with authentication, which the
// EMAIL task of a sending rule uses, and one on port 465 without
// authentication, used by no rule. The screens are visited by identifier.
func insertE2ESMTP(t *testing.T, db *database.DB) {
	t.Helper()

	for i, cred := range []*model.SMTPCredential{
		{EmailAddress: "alertes@passerelle.example", ServerAddress: types.Addr("smtp.example.org", 587),
			Login: "alertes", Password: "smtp-password-never-shown"},
		{EmailAddress: "rapports@passerelle.example", ServerAddress: types.Addr("smtp.example.org", 465)},
	} {
		require.NoError(t, db.Insert(cred).Run())
		require.EqualValues(t, i+1, cred.ID, "the screens are visited by identifier")
	}

	require.NoError(t, db.Insert(&model.EmailTemplate{Name: "fin-de-transfert", Subject: "Fin", Body: "Fichier reçu."}).Run())

	rule := &model.Rule{Name: "push-alertes", IsSend: true, Path: "push-alertes"}
	require.NoError(t, db.Insert(rule).Run())
	require.NoError(t, db.Insert(&model.Task{
		RuleID: rule.ID, Chain: model.ChainPost, Type: "EMAIL",
		Args: map[string]string{
			"sender": "alertes@passerelle.example", "recipients": "exploitation@client.example", "template": "fin-de-transfert",
		},
	}).Run())
}

// insertE2EKeys inserts an AES-128 key, which the tasks of a sending rule
// use to encrypt, and the PGP public key of a partner, used by no rule. The
// screens are visited by identifier.
func insertE2EKeys(t *testing.T, db *database.DB) {
	t.Helper()

	key, err := pgp.PGP().KeyGeneration().AddUserId("Partenaire", "partenaire@example.org").New().GenerateKey()
	require.NoError(t, err)

	public, err := key.GetArmoredPublicKey()
	require.NoError(t, err)

	for i, k := range []*model.CryptoKey{
		{Name: "aes-paie", Type: model.CryptoKeyTypeAES, Key: base64.StdEncoding.EncodeToString(make([]byte, 16))},
		{Name: "pgp-partenaire", Type: model.CryptoKeyTypePGPPublic, Key: public},
	} {
		require.NoError(t, db.Insert(k).Run())
		require.EqualValues(t, i+1, k.ID, "the screens are visited by identifier")
	}

	rule := &model.Rule{Name: "push-paie", IsSend: true, Path: "push-paie"}
	require.NoError(t, db.Insert(rule).Run())
	require.NoError(t, db.Insert(&model.Task{
		RuleID: rule.ID, Chain: model.ChainPre, Type: "ENCRYPT",
		Args: map[string]string{"keyName": "aes-paie", "method": "AES-CFB"},
	}).Run())
}

// insertE2ESNMP inserts an SNMPv2c monitor, an SNMPv3 monitor with
// authentication and encryption, and the configuration of the agent. The
// screens are visited by identifier.
func insertE2ESNMP(t *testing.T, db *database.DB) {
	t.Helper()

	for i, monitor := range []*snmp.MonitorConfig{
		{Name: "supervision-v2", Version: snmp.Version2, UDPAddress: "127.0.0.1:16162", Community: "smoke-community"},
		{
			Name: "supervision-v3", Version: snmp.Version3, UDPAddress: "127.0.0.1:16163",
			SNMPv3Security: snmp.V3SecurityAuthPriv, AuthEngineID: "0x80001f8880e9630000d61ff449",
			AuthUsername: "gw", AuthProtocol: snmp.AuthSHA256, AuthPassphrase: "smoke-auth-pass",
			PrivProtocol: snmp.PrivAES, PrivPassphrase: "smoke-priv-pass",
		},
	} {
		require.NoError(t, db.Insert(monitor).Run())
		require.EqualValues(t, i+1, monitor.ID, "the screens are visited by identifier")
	}

	require.NoError(t, db.Insert(&snmp.ServerConfig{LocalUDPAddress: "127.0.0.1:16161", SNMPv3Username: "agent"}).Run())
}

// insertE2EAuthorities inserts a TLS authority made for the test, valid two
// years, with an HTTPS partner whose certificate it signs (the R66 over TLS
// partner stays without means), and an SSH authority restricted to one SFTP
// server. The screens are visited by identifier.
func insertE2EAuthorities(t *testing.T, db *database.DB) {
	t.Helper()

	partner := &model.RemoteAgent{Name: "part-https", Protocol: "https", Address: types.Addr("127.0.0.3", 8443)}
	require.NoError(t, db.Insert(partner).Run())

	encode := func(der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	}

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Partners CA", Organization: []string{"Example"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(2, 0, 0),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "part-https"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(1, 0, 0),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP(partner.Address.Host)}, // the model checks the address of the partner
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caTmpl, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	sshPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	sshKey, err := ssh.NewPublicKey(sshPub)
	require.NoError(t, err)

	for i, authority := range []*model.Authority{
		{Name: "ca-partenaires", Type: "tls_authority", PublicIdentity: encode(caDER)},
		{
			Name: "ca-ssh-utilisateurs", Type: "ssh_cert_authority", PublicIdentity: string(ssh.MarshalAuthorizedKey(sshKey)),
			ValidHosts: []string{"sftp.example.org:22"},
		},
	} {
		require.NoError(t, db.Insert(authority).Run())
		require.EqualValues(t, i+1, authority.ID, "the screens are visited by identifier")
	}

	require.NoError(t, db.Insert(&model.Credential{
		RemoteAgentID: partner.NullableID(), Name: "cert-partenaire", Type: "trusted_tls_certificate", Value: encode(leafDER),
	}).Run())
}

// registerE2EStorages registers, under the type keys of two cloud
// instances, storages that never reach the network: the screens show the
// instances, the product instantiates the storage when one is saved.
func registerE2EStorages(t *testing.T) {
	t.Helper()

	for _, kind := range []string{"s3", "onedrive"} {
		fs.Register(kind, func(_, _, _ string, opts map[string]string) (fs.FS, error) {
			if kind == "onedrive" {
				opts["token"] = "e2e-token-never-shown"
			}

			return &fs.VFS{VFS: vfs.New(context.Background(), object.MemoryFs, &vfscommon.Options{})}, nil
		})
		t.Cleanup(func() { fs.Unregister(kind) })
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

	// After the default administrator of a fresh database (user 1): the
	// smoke user (user 2) and a viewer (user 3) the users screens show, edit
	// and delete; the screens are visited by identifier.
	for i, user := range []*model.User{
		{
			Username: e2eUsername, PasswordHash: string(hash),
			Permissions: model.PermServersRead | model.PermServersWrite | model.PermServersDelete |
				model.PermPartnersRead | model.PermPartnersWrite | model.PermPartnersDelete |
				model.PermUsersRead | model.PermUsersWrite | model.PermUsersDelete |
				model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete |
				model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete, // the SMTP credentials (#741), the email templates (#744)
		},
		{Username: "viewer", PasswordHash: string(hash), Permissions: model.PermServersRead | model.PermPartnersRead},
	} {
		require.NoError(t, db.Insert(user).Run())
		require.EqualValues(t, i+2, user.ID, "the screens are visited by identifier")
	}

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

	// The cloud instances (#719): an S3 bucket with a key and a secret, named
	// by the local folder of a rule; a OneDrive drive whose storage writes
	// its token in the options. The screens are visited by identifier.
	registerE2EStorages(t)

	for i, cloud := range []*model.CloudInstance{
		{Name: "archive-s3", Type: "s3", Key: "AKIAEXAMPLE", Secret: "s3-secret-never-shown",
			Options: map[string]string{"bucket": "archive", "region": "eu-west-3"}},
		{Name: "od-drive", Type: "onedrive", Key: "app-id", Secret: "app-secret-never-shown",
			Options: map[string]string{"tenant": "t-1", "drive_id": "d-1"}},
	} {
		require.NoError(t, db.Insert(cloud).Run())
		require.EqualValues(t, i+1, cloud.ID, "the screens are visited by identifier")
	}

	require.NoError(t, db.Insert(&model.Rule{Name: "pull-archive", IsSend: false, Path: "pull-archive", LocalDir: "archive-s3:in"}).Run())

	insertE2EAuthorities(t, db)
	insertE2ESNMP(t, db)
	insertE2EKeys(t, db)
	insertE2ESMTP(t, db)
	insertE2EEmailTemplates(t, db)

	// The gateway runs as a node of a cluster, with its own override file
	// and one address indirection (#730).
	db.Config.NodeID = "noeud-smoke"
	db.Config.Overrides = conf.NewOverride(filepath.Join(t.TempDir(), "noeud-smoke.ini"))
	require.NoError(t, db.Config.Overrides.AddIndirection("127.0.0.1", "10.0.0.5"))

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

		// A filter is a checkbox (the protocol families, the storage types)
		// or a select (the domain of the rights of a user): the first one is
		// changed.
		if s.count(t, `form[data-autosubmit] .w-disclose__panel input[type="checkbox"]`) > 0 {
			require.NoError(t, s.page.Locator(`form[data-autosubmit] .w-disclose__panel input[type="checkbox"]`).First().Check())
			require.NoError(t, s.tracker.settle())
			assert.Equal(t, 1, s.count(t, `form[data-autosubmit] .w-disclose__panel input[type="checkbox"]:checked`),
				"the filter is kept after the reload")
		} else {
			_, err := s.page.Locator(`form[data-autosubmit] .w-disclose__panel select`).First().
				SelectOption(playwright.SelectOptionValues{Indexes: playwright.IntSlice(1)})
			require.NoError(t, err)
			require.NoError(t, s.tracker.settle())
			assert.Equal(t, 1, s.count(t, `form[data-autosubmit] .w-disclose__panel select option:nth-child(2):checked`),
				"the filter is kept after the reload")
		}

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

	// The form of an email template (#744): typing "#" in the subject opens
	// the list of the variables, typing filters it and Enter inserts the
	// marker; the preview, computed by the server, follows the typing.
	if s.count(t, `form[data-preview]`) > 0 {
		subject := s.page.Locator(`#subject`)
		require.NoError(t, subject.Fill("Transfert "))
		require.NoError(t, subject.Press("End"))
		require.NoError(t, s.page.Keyboard().Type("#transferi"))
		first, err := s.page.Locator(`.w-vars [role="option"] .w-vars__code`).First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "#TRANSFERID#", first, "typing filters the variables, the name that starts with it first")
		require.NoError(t, s.page.Keyboard().Press("Enter"))

		value, err := subject.InputValue()
		require.NoError(t, err)
		assert.Equal(t, "Transfert #TRANSFERID#", value, "Enter inserts the marker")

		_, err = s.page.WaitForFunction(`document.querySelector("#et-preview-out").textContent.includes("Transfert 1234")`,
			nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)})
		require.NoError(t, err, "the preview follows the typing, with the example values")
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

	// The historic page of the authorities, which only pointed to the
	// command line, leads to the screen (#722).
	// The list then restores the state it was left in.
	t.Run("/webui/managing_authentication_authorities", func(t *testing.T) {
		session.visit(t, "/webui/managing_authentication_authorities")

		landed, err := url.Parse(page.URL())
		require.NoError(t, err)
		assert.Equal(t, "/webui/authorities", landed.Path)
	})

	// So does the historic page of SNMP (#723).
	t.Run("/webui/snmp_management", func(t *testing.T) {
		session.visit(t, "/webui/snmp_management")

		landed, err := url.Parse(page.URL())
		require.NoError(t, err)
		assert.Equal(t, "/webui/snmp", landed.Path)
	})

	// And the historic page of the configuration overrides (#730).
	t.Run("/webui/managing_configuration_overrides", func(t *testing.T) {
		session.visit(t, "/webui/managing_configuration_overrides")

		landed, err := url.Parse(page.URL())
		require.NoError(t, err)
		assert.Equal(t, "/webui/overrides", landed.Path)
	})

	// And the historic page of the cryptographic keys (#735).
	t.Run("/webui/cryptographic_key_management", func(t *testing.T) {
		session.visit(t, "/webui/cryptographic_key_management")

		landed, err := url.Parse(page.URL())
		require.NoError(t, err)
		assert.Equal(t, "/webui/keys", landed.Path)
	})

	// And the historic page of the SMTP credentials (#741).
	t.Run("/webui/smtp_credentials_management", func(t *testing.T) {
		session.visit(t, "/webui/smtp_credentials_management")

		landed, err := url.Parse(page.URL())
		require.NoError(t, err)
		assert.Equal(t, "/webui/smtp", landed.Path)
	})

	// And the historic page of the email templates (#744).
	t.Run("/webui/email_templates_management", func(t *testing.T) {
		session.visit(t, "/webui/email_templates_management")

		landed, err := url.Parse(page.URL())
		require.NoError(t, err)
		assert.Equal(t, "/webui/email-templates", landed.Path)
	})
}
