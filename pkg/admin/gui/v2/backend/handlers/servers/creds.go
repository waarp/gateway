package servers

// The authentication means of a server (its identity) or of a local account
// (what the partner presents): the "Authentication" tab, the page of a means,
// the form that adds or edits one and the confirmation of a deletion. The
// screen never shows a secret: a certificate is summarised by its subject,
// its validity and its fingerprint, a key by its kind and the fingerprint of
// its public part, a password by "set". On edition, an empty secret keeps the
// current one.

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"slices"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
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

// Families of means: the block of fields of the form and the shape of the
// summary.
const (
	familyPassword = "password" // one secret
	familyFile     = "file"     // one file: a certificate, a public key or a private key
	familyTwoFile  = "twofile"  // a certificate and its private key
	familyNone     = "none"     // nothing to give: the kind itself is the information

	// Names of the fields of the form of a means, prefixed inside the form
	// of an account ("means_type").
	fieldType     = "type"
	fieldPassword = "password"
	fieldFile     = "file"
	fieldPem      = "pem"
	fieldCertFile = "cert_file"
	fieldCertPem  = "cert_pem"
	fieldKeyFile  = "key_file"
	fieldKeyPem   = "key_pem"
	meansPrefix   = "means_"

	// The validity state of a certificate: the triangle of the kit for one
	// that expires soon.
	stateWarn     = "warn"
	expiryWarning = 30 * 24 * time.Hour

	typeKeyPrefix = "auth.type."
)

//nolint:gochecknoglobals // reference data of the screen
var (
	// The kinds of means the screen handles, by the keys
	// the model registers.
	credFamilies = map[string]string{
		auth.Password:                 familyPassword,
		auth.TLSCertificate:           familyTwoFile,
		auth.TLSTrustedCertificate:    familyFile,
		sftp.AuthSSHPrivateKey:        familyFile,
		sftp.AuthSSHPublicKey:         familyFile,
		r66auth.AuthLegacyCertificate: familyNone,
	}

	credsCounter = counter{"auth.count_one", "auth.count", "auth.count_shown_one", "auth.count_shown"}
)

func familyOf(typ string) string { return credFamilies[typ] }

// typeLabel names a kind of means; a kind the screen does not know keeps its key.
func typeLabel(translate gui.Translator, typ string) string {
	if _, ok := credFamilies[typ]; ok {
		return translate(typeKeyPrefix + typ)
	}

	return typ
}

// serverCredTypes are the means a server of the protocol presents to the
// partners (its identity): the table of the historic page, without the
// PeSIT pre-connection that the model refuses on a server. The legacy R66
// certificate is offered when the instance allows it.
func serverCredTypes(proto string) []string {
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

// accountCredTypes are the means a partner may present on an account of a
// server of the protocol: a password always, a trusted certificate over
// TLS, a public key over SSH, the legacy R66 certificate when the instance
// allows it.
func accountCredTypes(proto string) []string {
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

func withLegacy(types []string) []string {
	if compatibility.IsLegacyR66CertificateAllowed {
		return append(types, r66auth.AuthLegacyCertificate)
	}

	return types
}

// unique tells whether the owner may hold a single means of the kind, from
// the model's own handler of the kind.
func unique(own owner, typ string) bool {
	var handler authentication.Handler
	if own.isAccount() {
		handler = authentication.GetInternalAuthHandler(typ, own.srv.agent.Protocol)
	} else {
		handler = authentication.GetExternalAuthMethod(typ, own.srv.agent.Protocol)
	}

	return handler != nil && handler.CanOnlyHaveOne()
}

// remainingTypes are the kinds one can still add to the owner: a unique
// kind already present is left out.
func remainingTypes(own owner, creds []*model.Credential) []string {
	var out []string

	for _, typ := range own.allowed() {
		present := slices.ContainsFunc(creds, func(c *model.Credential) bool { return c.Type == typ })
		if !present || !unique(own, typ) {
			out = append(out, typ)
		}
	}

	return out
}

// ------------------------------------------------------------ credView ----

// credView is a means as the screen shows it: never the secret, only what
// identifies it.
type credView struct {
	ID          int64
	Name        string
	Type        string
	Subject     string
	Issuer      string
	SelfSigned  bool
	Fingerprint string
	KeyType     string
	NotBefore   time.Time
	NotAfter    time.Time
	Public      string // the public part, shareable: a certificate PEM, a public key line
}

func viewOf(c *model.Credential) credView {
	view := credView{ID: c.ID, Name: c.Name, Type: c.Type}

	switch c.Type {
	case auth.TLSCertificate, auth.TLSTrustedCertificate:
		view.readCertificate(c.Value)
	case sftp.AuthSSHPrivateKey:
		if signer, err := ssh.ParsePrivateKey([]byte(c.Value)); err == nil {
			view.readKey(signer.PublicKey())
		}
	case sftp.AuthSSHPublicKey:
		if key, err := sftp.ParseAuthorizedKey(c.Value); err == nil {
			view.readKey(key)
			view.Public = strings.TrimSpace(c.Value)
		}
	}

	return view
}

// readCertificate keeps the subject, the issuer, the validity and the
// SHA-256 fingerprint of a certificate; one that cannot be read leaves
// nothing, and the means is summarised as "set".
func (v *credView) readCertificate(pemText string) {
	cert, err := auth.ParseCertPEM(pemText)
	if err != nil {
		return
	}

	sum := sha256.Sum256(cert.Raw)
	v.Subject = subjectOf(cert)
	v.Issuer = cert.Issuer.CommonName
	v.SelfSigned = bytes.Equal(cert.RawIssuer, cert.RawSubject)
	v.NotBefore, v.NotAfter = cert.NotBefore, cert.NotAfter
	v.Fingerprint = colons(sum[:])
	v.Public = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
}

// subjectOf names a certificate: its common name, or the first name it is
// issued for, or its whole subject.
func subjectOf(cert *x509.Certificate) string {
	switch {
	case cert.Subject.CommonName != "":
		return cert.Subject.CommonName
	case len(cert.DNSNames) > 0:
		return cert.DNSNames[0]
	case len(cert.IPAddresses) > 0:
		return cert.IPAddresses[0].String()
	default:
		return cert.Subject.String()
	}
}

// colons writes a fingerprint as the certificate tools do: upper-case hex
// pairs separated by colons.
func colons(sum []byte) string {
	const pair = 2 // hex digits per byte

	h := strings.ToUpper(hex.EncodeToString(sum))
	pairs := make([]string, 0, len(h)/pair)

	for i := 0; i+1 < len(h); i += pair {
		pairs = append(pairs, h[i:i+pair])
	}

	return strings.Join(pairs, ":")
}

func (v *credView) readKey(key ssh.PublicKey) {
	v.KeyType = key.Type()
	v.Fingerprint = ssh.FingerprintSHA256(key)
}

func (v *credView) isCert() bool {
	return v.Type == auth.TLSCertificate || v.Type == auth.TLSTrustedCertificate
}

// expiry is the validity state of a certificate: expired, expiring within
// thirty days, or valid; nothing for the other kinds.
func (v *credView) expiry(translate gui.Translator, now time.Time) gui.Status {
	if !v.isCert() || v.NotAfter.IsZero() {
		return gui.Status{}
	}

	date := v.NotAfter.Format(translate("auth.date"))

	switch {
	case v.NotAfter.Before(now):
		return gui.Status{State: stateKO, Label: translate("auth.content.expired", "date", date)}
	case v.NotAfter.Before(now.Add(expiryWarning)):
		return gui.Status{State: stateWarn, Label: translate("auth.content.expires", "date", date)}
	default:
		return gui.Status{State: stateOK, Label: translate("auth.content.valid", "date", date)}
	}
}

// summary is the non-secret content shown in the list.
func (v *credView) summary(translate gui.Translator) string {
	switch {
	case v.Type == auth.Password:
		return translate("auth.content.secret")
	case v.Type == auth.TLSCertificate && v.Subject != "":
		return translate("auth.content.cert_key", "subject", v.Subject)
	case v.Type == auth.TLSTrustedCertificate && v.Subject != "":
		return v.Subject
	case v.Type == sftp.AuthSSHPrivateKey && v.KeyType != "":
		return translate("auth.content.private_key", "type", v.KeyType)
	case v.Type == sftp.AuthSSHPublicKey && v.KeyType != "":
		return translate("auth.content.key", "type", v.KeyType, "fingerprint", v.Fingerprint)
	default:
		return translate("auth.content.set")
	}
}

// ---------------------------------------------------------------- list ----

type credRow struct {
	ID      int64
	Name    string
	Type    string
	Summary string
	Expiry  gui.Status
	URL     string
	Actions []gui.Action
}

type credListData struct {
	List         tabList
	Title        string
	Intro        string
	Tabs         []tab
	BackURL      string
	BackLabel    string
	AddURL       string // empty without the write right or when nothing can be added
	NonePossible bool   // the protocol asks no means from the owner
	EmptyText    string
	Rows         []credRow
}

// credActions is the "⋯" menu of a means: view, edit (write right), delete
// (delete right).
func credActions(translate gui.Translator, own owner, id int64, user rights, carried string) []gui.Action {
	acts := []gui.Action{{Icon: "eye", Label: translate("auth.action.detail"), URL: withRet(own.credURL(id, ""), carried)}}

	if user.Write {
		acts = append(acts, gui.Action{
			Icon: "pen", Label: translate("auth.action.edit"), URL: withRet(own.credURL(id, editSuffix), carried),
		})
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("auth.action.delete"),
			URL:    withRet(own.credURL(id, deleteSuffix), carried),
			Danger: true,
		})
	}

	return acts
}

// buildCredList computes the "Authentication" tab of an owner: the means
// searched by name, sorted, paged, each one with its non-secret summary.
func buildCredList(translate gui.Translator, own owner, creds []*model.Credential, query listQuery,
	user rights, carried string, now time.Time,
) credListData {
	base := own.authURL()
	data := credListData{
		List:      newTabList(base, query, carried),
		Title:     own.title(translate),
		Intro:     translate("auth.title.server"),
		BackURL:   listURL(carried),
		BackLabel: translate("servers.back"),
		EmptyText: translate("auth.empty.server"),
	}

	if own.isAccount() {
		data.Intro = translate("auth.title.account")
		data.BackURL = withRet(serverURL(own.srv.agent.ID, accountsSuffix), carried)
		data.BackLabel = translate("servers.back_server")
		data.EmptyText = translate("auth.empty.account")
	}

	data.NonePossible = len(own.allowed()) == 0

	if user.Write && len(remainingTypes(own, creds)) > 0 {
		data.AddURL = withRet(base+newSuffix, carried)
	}

	views := make([]credView, 0, len(creds))
	needle := strings.ToLower(query.Q)

	for _, c := range creds {
		data.List.Names = append(data.List.Names, c.Name)

		if needle == "" || strings.HasPrefix(strings.ToLower(c.Name), needle) {
			views = append(views, viewOf(c))
		}
	}

	slices.Sort(data.List.Names)
	sortViews(translate, views, query.Sort, query.Dir)

	data.List.Count = credsCounter.text(translate, len(creds), len(views))

	switch {
	case len(creds) == 0:
		data.List.Empty = true

		return data
	case len(views) == 0:
		data.List.noMatch(translate, translate("auth.none.title", "query", query.Q), base, carried)

		return data
	}

	b := pageOf(len(views), query)
	data.List.Caption = translate("auth.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate("auth.sort."+query.Sort))

	for i := b.From; i < b.To; i++ {
		view := &views[i]
		data.Rows = append(data.Rows, credRow{
			ID:      view.ID,
			Name:    view.Name,
			Type:    typeLabel(translate, view.Type),
			Summary: view.summary(translate),
			Expiry:  view.expiry(translate, now),
			URL:     withRet(own.credURL(view.ID, ""), carried),
			Actions: credActions(translate, own, view.ID, user, carried),
		})
	}

	data.List.foot(translate("auth.pager", "from", b.From+1, "to", b.To, "total", len(views)), b, query, base, carried)

	return data
}

// sortViews sorts by name or by kind; ties keep the name order.
func sortViews(translate gui.Translator, views []credView, key, dir string) {
	byName := func(a, b credView) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }

	slices.SortStableFunc(views, func(a, b credView) int {
		order := byName(a, b)
		if key == sortType {
			order = strings.Compare(typeLabel(translate, a.Type), typeLabel(translate, b.Type))
		}

		if dir == dirDesc {
			order = -order
		}

		if order == 0 {
			return byName(a, b)
		}

		return order
	})
}

// Auth serves GET /webui/servers/{id}/auth and GET
// /webui/servers/{id}/accounts/{aid}/auth: the means of the owner.
func (h *Handlers) Auth(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	translate := h.auth.Translator(r)
	carried := ret(r)

	creds, err := own.table().GetCredentials(h.db)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := buildCredList(translate, own, creds, parseQuery(r.URL.Query(), credSortKeys), rightsOf(r), carried, time.Now())

	if data.Tabs, err = h.tabsOf(translate, own, tabAuth, carried); err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	screen := h.sheetScreen(translate, data.Title, own, "", carried, 0, data, notices(translate, r.URL.Query()))
	h.auth.Render(w, h.logger, r, screen)
}

// ---------------------------------------------------------------- form ----

// credFormData is the form of a means: its identity (name, kind) and the
// content of the kind. Inside the form of an account, the fields are
// prefixed and the kind may be left empty ("none for now").
type credFormData struct {
	Title     string
	Action    string
	CancelURL string
	Owner     string
	Prefix    string
	Edit      bool
	Optional  bool
	HasErrors bool
	Refused   string
	Tabs      []tab

	Name      field
	Type      string
	TypeLabel string
	TypeError string
	Family    string
	Types     []choice
	Password  field
	Pem       field
	PemCert   field
	PemKey    field

	// What is in place, on edition: an empty field keeps it.
	CurrentFile     string
	CurrentCert     string
	CurrentKey      string
	CurrentPassword string
}

// typeChoices lists the kinds offered: with "none for now" when the means is
// optional, with "choose a kind" until one is chosen otherwise.
func typeChoices(translate gui.Translator, kinds []string, current string, optional bool) []choice {
	out := make([]choice, 0, len(kinds)+1)

	switch {
	case optional:
		out = append(out, choice{Value: "", Label: translate("accounts.form.first_means_none"), Selected: current == ""})
	case current == "":
		out = append(out, choice{Value: "", Label: translate("auth.form.type_choose"), Selected: true})
	}

	for _, typ := range kinds {
		out = append(out, choice{Value: typ, Label: typeLabel(translate, typ), Selected: typ == current})
	}

	return out
}

// credForm builds the form of a new means of the owner, or the form of the
// given one filled with what identifies it.
func (h *Handlers) newCredForm(translate gui.Translator, own owner, cred *model.Credential,
	creds []*model.Credential, carried string, now time.Time,
) *credFormData {
	f := &credFormData{Owner: own.label(translate)}

	if cred == nil {
		f.Title = translate("auth.new.title")
		f.Action = withRet(own.authURL()+newSuffix, carried)
		f.CancelURL = withRet(own.authURL(), carried)
		f.Types = typeChoices(translate, remainingTypes(own, creds), "", false)

		return f
	}

	f.Title = translate("auth.edit.title", "name", cred.Name)
	f.Action = withRet(own.credURL(cred.ID, editSuffix), carried)
	f.CancelURL = withRet(own.credURL(cred.ID, ""), carried)
	f.Edit = true
	f.Name = field{Value: cred.Name}
	f.Type, f.TypeLabel, f.Family = cred.Type, typeLabel(translate, cred.Type), familyOf(cred.Type)
	view := viewOf(cred)
	f.current(translate, &view, now)

	return f
}

// current says what is in place next to each field of the kind, so that the
// operator knows what an empty field keeps.
func (f *credFormData) current(translate gui.Translator, view *credView, now time.Time) {
	switch view.Type {
	case auth.TLSCertificate:
		f.CurrentCert = certText(translate, view, now)
		f.CurrentKey = translate("auth.content.key_set")
	case auth.TLSTrustedCertificate:
		f.CurrentFile = certText(translate, view, now)
	case sftp.AuthSSHPrivateKey, sftp.AuthSSHPublicKey:
		f.CurrentFile = view.summary(translate)
	case auth.Password:
		f.CurrentPassword = translate("auth.content.secret")
	}
}

// certText summarises a certificate in place: its subject and its validity.
func certText(translate gui.Translator, view *credView, now time.Time) string {
	if view.Subject == "" {
		return translate("auth.content.set")
	}

	return view.Subject + " · " + view.expiry(translate, now).Label
}

// read takes the posted fields into the form and the means: the name, the
// kind (at creation) and the content of the family of the kind. On edition,
// an empty secret keeps the current one. The kind is checked against what
// the owner may still receive; the model checks everything else when the
// means is saved.
func (f *credFormData) read(translate gui.Translator, r *http.Request, own owner,
	cred *model.Credential, creds []*model.Credential,
) error {
	p := f.Prefix
	f.Name.Value = strings.TrimSpace(r.FormValue(p + fieldName))

	if !f.Edit {
		f.Type = r.FormValue(p + fieldType)
		f.Family = familyOf(f.Type)
		f.TypeLabel = typeLabel(translate, f.Type)
		cred.Type = f.Type

		switch {
		case slices.Contains(remainingTypes(own, creds), f.Type):
		case slices.Contains(own.allowed(), f.Type):
			f.TypeError = translate("auth.form.error.unique")
		default:
			f.TypeError = translate("auth.form.error.type")
		}
	}

	cred.Name = f.Name.Value
	if cred.Name == "" {
		cred.Name = f.TypeLabel
	}

	var err error

	switch f.Family {
	case familyPassword:
		f.readPassword(translate, r, cred)
	case familyFile:
		err = f.readFile(translate, r, cred)
	case familyTwoFile:
		err = f.readPair(translate, r, cred)
	}

	if err != nil {
		return err
	}

	f.HasErrors = f.TypeError != "" || f.Password.Error != "" || f.Pem.Error != "" ||
		f.PemCert.Error != "" || f.PemKey.Error != ""

	return nil
}

// readPassword takes the password; empty, it is required at creation and
// kept on edition.
func (f *credFormData) readPassword(translate gui.Translator, r *http.Request, cred *model.Credential) {
	if password := r.FormValue(f.Prefix + fieldPassword); password != "" {
		cred.Value = password
	} else if !f.Edit {
		f.Password.Error = translate("auth.form.error.required")
	}
}

// readFile takes the one file of the means, uploaded or pasted. A pasted
// private key is not echoed back on a refused form; the public contents are.
func (f *credFormData) readFile(translate gui.Translator, r *http.Request, cred *model.Credential) error {
	content, err := fileOrText(r, f.Prefix+fieldFile, f.Prefix+fieldPem)
	if err != nil {
		return err
	}

	if f.Type != sftp.AuthSSHPrivateKey {
		f.Pem.Value = r.FormValue(f.Prefix + fieldPem)
	}

	if content != "" {
		cred.Value = content
	} else if !f.Edit {
		f.Pem.Error = translate("auth.form.error.required")
	}

	return nil
}

// readPair takes the certificate and its private key, each one uploaded or
// pasted, each one kept on edition when left empty.
func (f *credFormData) readPair(translate gui.Translator, r *http.Request, cred *model.Credential) error {
	cert, err := fileOrText(r, f.Prefix+fieldCertFile, f.Prefix+fieldCertPem)
	if err != nil {
		return err
	}

	key, err := fileOrText(r, f.Prefix+fieldKeyFile, f.Prefix+fieldKeyPem)
	if err != nil {
		return err
	}

	f.PemCert.Value = r.FormValue(f.Prefix + fieldCertPem)
	required := translate("auth.form.error.required")

	if cert != "" {
		cred.Value = cert
	} else if !f.Edit {
		f.PemCert.Error = required
	}

	if key != "" {
		cred.Value2 = key
	} else if !f.Edit {
		f.PemKey.Error = required
	}

	return nil
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *credFormData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "same name"):
		f.Name.Error = translate("auth.form.error.name_taken")
	case strings.Contains(msg, "already has a"):
		f.TypeError = translate("auth.form.error.unique")
	case strings.Contains(msg, "failed to validate authentication value"):
		_, reason, _ := strings.Cut(msg, "failed to validate authentication value: ")
		text := translate("auth.form.error.invalid", "reason", reason)

		switch f.Family {
		case familyPassword:
			f.Password.Error = text
		case familyFile:
			f.Pem.Error = text
		case familyTwoFile:
			f.PemCert.Error = text
		default:
			f.Refused = text
		}
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewCredForm serves GET …/auth/new.
func (h *Handlers) NewCredForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	creds, err := own.table().GetCredentials(h.db)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	translate := h.credForm.Translator(r)
	carried := ret(r)
	h.credFormPage(w, r, own, h.newCredForm(translate, own, nil, creds, carried, time.Now()), 0)
}

// CreateCred serves POST …/auth/new.
func (h *Handlers) CreateCred(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	h.saveCred(w, r, own, nil)
}

// EditCredForm serves GET …/auth/{cid}/edit.
func (h *Handlers) EditCredForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	cred, ok := h.credential(w, r, own)
	if !ok {
		return
	}

	translate := h.credForm.Translator(r)
	carried := ret(r)
	h.credFormPage(w, r, own, h.newCredForm(translate, own, cred, nil, carried, time.Now()), 0)
}

// UpdateCred serves POST …/auth/{cid}/edit.
func (h *Handlers) UpdateCred(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	cred, ok := h.credential(w, r, own)
	if !ok {
		return
	}

	h.saveCred(w, r, own, cred)
}

// credFormPage renders the form of a means with the tab strip of its owner.
func (h *Handlers) credFormPage(w http.ResponseWriter, r *http.Request, own owner, f *credFormData, status int) {
	translate := h.credForm.Translator(r)
	carried := ret(r)

	tabs, err := h.tabsOf(translate, own, tabAuth, carried)
	if err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	f.Tabs = tabs
	h.credForm.Render(w, h.logger, r, h.sheetScreen(translate, f.Title, own, f.Title, carried, status, f, nil))
}

// saveCred reads, validates and stores the posted means: a new one when cred
// is nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects to the "Authentication" tab of the
// owner with a notification.
func (h *Handlers) saveCred(w http.ResponseWriter, r *http.Request, own owner, cred *model.Credential) {
	translate := h.credForm.Translator(r)
	carried := ret(r)

	if err := parseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	creds, err := own.table().GetCredentials(h.db)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	form := h.newCredForm(translate, own, cred, creds, carried, time.Now())
	done := doneUpdated

	if cred == nil {
		cred = &model.Credential{}
		own.table().SetCredOwner(cred)
		done = doneCreated
	}

	if readErr := form.read(translate, r, own, cred, creds); readErr != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	if !form.HasErrors {
		var saveErr error
		if done == doneCreated {
			saveErr = h.db.Insert(cred).Run()
		} else {
			saveErr = h.db.Update(cred).Run()
		}

		switch {
		case saveErr == nil:
			redirectSheet(w, r, withRet(own.authURL(), carried), kindCred, done, cred.Name)

			return
		case database.IsValidationError(saveErr):
			form.refuse(translate, saveErr)
		default:
			h.logger.Errorf("failed to save the authentication means %q: %v", cred.Name, saveErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	if !form.Edit {
		form.Types = typeChoices(translate, remainingTypes(own, creds), form.Type, false)
	}

	h.credFormPage(w, r, own, form, http.StatusUnprocessableEntity)
}

// -------------------------------------------------------------- detail ----

type credDetailData struct {
	Title       string
	Name        string
	Owner       string
	Type        string
	Tabs        []tab
	BackURL     string
	EditURL     string
	DeleteURL   string
	Expiry      gui.Status
	Rows        []kv
	Public      string // the public part, when there is one
	PublicTitle string
}

// buildCredDetail describes a means in reading: what identifies it, never
// the secret.
func buildCredDetail(translate gui.Translator, own owner, view *credView, user rights, carried string,
	now time.Time,
) credDetailData {
	data := credDetailData{
		Title:   translate("auth.detail.title", "name", view.Name),
		Name:    view.Name,
		Owner:   own.label(translate),
		Type:    typeLabel(translate, view.Type),
		BackURL: withRet(own.authURL(), carried),
		Expiry:  view.expiry(translate, now),
	}

	if user.Write {
		data.EditURL = withRet(own.credURL(view.ID, editSuffix), carried)
	}

	if user.Delete {
		data.DeleteURL = withRet(own.credURL(view.ID, deleteSuffix), carried)
	}

	never := translate("auth.detail.never_shown")
	date := translate("auth.date")

	switch {
	case view.isCert() && view.Subject != "":
		issuer := kv{Label: translate("auth.detail.issuer"), Value: view.Issuer, Mono: true}
		if view.SelfSigned {
			issuer = kv{Label: translate("auth.detail.issuer"), Value: translate("auth.detail.self_signed")}
		}

		data.Rows = []kv{
			{Label: translate("auth.detail.subject"), Value: view.Subject, Mono: true},
			issuer,
			{
				Label: translate("auth.detail.validity"),
				Value: translate("auth.detail.validity_text",
					"from", view.NotBefore.Format(date), "to", view.NotAfter.Format(date)),
			},
			{Label: translate("auth.detail.fingerprint"), Value: view.Fingerprint, Mono: true},
		}

		if view.Type == auth.TLSCertificate {
			data.Rows = append(data.Rows, kv{Label: translate("auth.detail.private_key"), Value: never})
		}

		data.Public, data.PublicTitle = view.Public, translate("auth.detail.pem")
	case view.Type == sftp.AuthSSHPublicKey && view.KeyType != "":
		data.Rows = []kv{
			{Label: translate("auth.detail.key_type"), Value: view.KeyType, Mono: true},
			{Label: translate("auth.detail.fingerprint"), Value: view.Fingerprint, Mono: true},
		}
		data.Public, data.PublicTitle = view.Public, translate("auth.detail.public_key")
	case view.Type == sftp.AuthSSHPrivateKey && view.KeyType != "":
		data.Rows = []kv{
			{Label: translate("auth.detail.key_type"), Value: view.KeyType, Mono: true},
			{Label: translate("auth.detail.fingerprint"), Value: view.Fingerprint, Mono: true},
			{Label: translate("auth.detail.private_key"), Value: never},
		}
	case view.Type == r66auth.AuthLegacyCertificate:
		data.Rows = []kv{{Label: translate("auth.detail.content"), Value: translate("auth.detail.legacy")}}
	default:
		data.Rows = []kv{{Label: translate("auth.detail.content"), Value: never}}
	}

	return data
}

// CredDetail serves GET …/auth/{cid}.
func (h *Handlers) CredDetail(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	cred, ok := h.credential(w, r, own)
	if !ok {
		return
	}

	translate := h.credDetail.Translator(r)
	carried := ret(r)
	view := viewOf(cred)
	data := buildCredDetail(translate, own, &view, rightsOf(r), carried, time.Now())

	tabs, err := h.tabsOf(translate, own, tabAuth, carried)
	if err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data.Tabs = tabs
	h.credDetail.Render(w, h.logger, r, h.sheetScreen(translate, data.Title, own, cred.Name, carried, 0, data, nil))
}

// -------------------------------------------------------------- delete ----

type credDeleteData struct {
	Title     string
	Text      string
	Name      string
	Type      string
	Owner     string
	Summary   string
	Tabs      []tab
	CancelURL string
	Action    string
}

func (h *Handlers) credDeletePage(w http.ResponseWriter, r *http.Request, own owner, cred *model.Credential) {
	translate := h.credDel.Translator(r)
	carried := ret(r)
	view := viewOf(cred)

	data := credDeleteData{
		Title:     translate("auth.delete.title", "name", cred.Name),
		Text:      translate("auth.delete.server"),
		Name:      cred.Name,
		Type:      typeLabel(translate, cred.Type),
		Owner:     own.label(translate),
		Summary:   view.summary(translate),
		CancelURL: withRet(own.credURL(cred.ID, ""), carried),
		Action:    withRet(own.credURL(cred.ID, deleteSuffix), carried),
	}

	if own.isAccount() {
		data.Text = translate("auth.delete.account")
	}

	tabs, err := h.tabsOf(translate, own, tabAuth, carried)
	if err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data.Tabs = tabs
	h.credDel.Render(w, h.logger, r, h.sheetScreen(translate, data.Title, own, data.Title, carried, 0, data, nil))
}

// DeleteCredForm serves GET …/auth/{cid}/delete: the confirmation page.
func (h *Handlers) DeleteCredForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	cred, ok := h.credential(w, r, own)
	if !ok {
		return
	}

	h.credDeletePage(w, r, own, cred)
}

// DeleteCred serves POST …/auth/{cid}/delete.
func (h *Handlers) DeleteCred(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	cred, ok := h.credential(w, r, own)
	if !ok {
		return
	}

	if err := h.db.Delete(cred).Run(); err != nil {
		h.logger.Errorf("failed to delete the authentication means %q: %v", cred.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	redirectSheet(w, r, withRet(own.authURL(), ret(r)), kindCred, doneDeleted, cred.Name)
}
