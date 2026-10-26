package means

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certchain"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

// Names of the fields of the form of a means, prefixed inside the form of
// an account ("means_type").
const (
	fieldName     = "name"
	fieldType     = "type"
	fieldPassword = "password"
	fieldFile     = "file"
	fieldPem      = "pem"
	fieldCertFile = "cert_file"
	fieldCertPem  = "cert_pem"
	fieldKeyFile  = "key_file"
	fieldKeyPem   = "key_pem"
	// The PeSIT pre-connection: a login and a password, both optional.
	fieldPreLogin    = "pre_login"
	fieldPrePassword = "pre_password"

	// A PEM file weighs a few kilobytes: the limit of an upload.
	maxUpload = 4 << 20

	// The causes the product gives, worded for a developer, that the form
	// says in the words of the user: a chain that reaches no authority the
	// gateway knows; a certificate given in place of the private key, a key
	// that is not the one of the certificate, a key that cannot be read.
	causeUnknownAuthority = "signed by unknown authority"
	causeKeyIsCert        = "found a certificate rather than a key"
	causeKeyMismatch      = "does not match public key"
	causeKeyUnreadable    = "failed to parse private key"
	causeKeyEmpty         = "failed to find any PEM data in key input"
)

// Form is the form of a means: its identity (name, kind) and the content of
// the kind. Inside the form of an account, the fields are prefixed and the
// kind may be left empty ("none for now").
type Form struct {
	Title     string
	Action    string
	CancelURL string
	Owner     string
	Prefix    string
	Edit      bool
	Optional  bool
	HasErrors bool
	Refused   string
	Tabs      []Tab
	TabsLabel string
	// After the save, Note says what the form did with the certificates
	// given beyond the chain: noteRootDropped (Root named), or
	// noteTrustedLeafOnly.
	Note string
	Root string
	// chain is the chain assembled from the certificates given, for the
	// explanation of a refusal of the product.
	chain *certchain.Chain

	Name      protoform.Field
	Type      string
	TypeLabel string
	TypeError string
	Family    string
	Types     []protoform.Choice
	Password  protoform.Field
	Pem       protoform.Field
	PemCert   protoform.Field
	PemKey    protoform.Field
	// The PeSIT pre-connection: the login is shown again, the password never.
	PreLogin    protoform.Field
	PrePassword protoform.Field

	// What is in place, on edition: an empty field keeps it.
	CurrentFile     string
	CurrentCert     string
	CurrentKey      string
	CurrentPassword string
}

// TypeChoices lists the kinds offered: with "none for now" when the means is
// optional, with "choose a kind" until one is chosen otherwise.
func TypeChoices(translate gui.Translator, kinds []string, current string, optional bool) []protoform.Choice {
	out := make([]protoform.Choice, 0, len(kinds)+1)

	switch {
	case optional:
		out = append(out, protoform.Choice{
			Value: "", Label: translate("accounts.form.first_means_none"), Selected: current == "",
		})
	case current == "":
		out = append(out, protoform.Choice{Value: "", Label: translate("auth.form.type_choose"), Selected: true})
	}

	for _, typ := range kinds {
		out = append(out, protoform.Choice{Value: typ, Label: TypeLabel(translate, typ), Selected: typ == current})
	}

	return out
}

// FirstMeans is the form of the first means of an account, inside the form
// of the account: prefixed fields, kind optional among the given ones.
func FirstMeans(translate gui.Translator, kinds []string, current string) *Form {
	return &Form{Prefix: Prefix, Optional: true, Types: TypeChoices(translate, kinds, current, true)}
}

// newForm builds the form of a new means of the owner, or the form of the
// given one filled with what identifies it.
func newForm(translate gui.Translator, own *Owner, cred *model.Credential, creds []*model.Credential,
	now time.Time,
) *Form {
	f := &Form{Owner: own.Label, TabsLabel: translate(own.Texts.Tabs)}

	if cred == nil {
		f.Title = translate("auth.new.title")
		f.Action = own.WithRet(own.AuthURL + NewSuffix)
		f.CancelURL = own.WithRet(own.AuthURL)
		f.Types = TypeChoices(translate, own.Remaining(creds), "", false)

		return f
	}

	f.Title = translate("auth.edit.title", "name", cred.Name)
	f.Action = own.WithRet(own.CredURL(cred.ID, EditSuffix))
	f.CancelURL = own.WithRet(own.CredURL(cred.ID, ""))
	f.Edit = true
	f.Name = protoform.Field{Value: cred.Name}
	f.Type, f.TypeLabel, f.Family = cred.Type, TypeLabel(translate, cred.Type), FamilyOf(cred.Type)
	v := viewOf(cred)
	f.current(translate, &v, now)

	return f
}

// current says what is in place next to each field of the kind, so that the
// operator knows what an empty field keeps.
func (f *Form) current(translate gui.Translator, v *view, now time.Time) {
	switch v.Type {
	case auth.TLSCertificate:
		f.CurrentCert = certText(translate, v, now)
		f.CurrentKey = translate("auth.content.key_set")
	case auth.TLSTrustedCertificate:
		f.CurrentFile = certText(translate, v, now)
	case sftp.AuthSSHPrivateKey, sftp.AuthSSHPublicKey:
		f.CurrentFile = v.summary(translate)
	case auth.Password:
		f.CurrentPassword = translate("auth.content.secret")
	case pesit.PreConnectionAuth:
		f.PreLogin.Value = v.Login
		f.CurrentPassword = translate("auth.content.pre_no_password")

		if v.HasPassword {
			f.CurrentPassword = translate("auth.content.pre_password")
		}
	}
}

// certText summarises a certificate in place: its subject and its validity.
func certText(translate gui.Translator, v *view, now time.Time) string {
	if v.Subject == "" {
		return translate("auth.content.set")
	}

	return v.Subject + v.chainText(translate) + " · " + v.expiry(translate, now).Label
}

// Chosen tells whether a kind was posted: inside the form of an account,
// the first means is optional.
func (f *Form) Chosen(r *http.Request) bool { return r.FormValue(f.Prefix+fieldType) != "" }

// Read takes the posted fields into the form and the means: the name, the
// kind (at creation) and the content of the family of the kind. On edition,
// an empty secret keeps the current one. The kind is checked against what
// the owner may still receive; the model checks everything else when the
// means is saved.
func (f *Form) Read(translate gui.Translator, r *http.Request, own *Owner,
	cred *model.Credential, creds []*model.Credential,
) error {
	p := f.Prefix
	f.Name.Value = strings.TrimSpace(r.FormValue(p + fieldName))

	if !f.Edit {
		f.Type = r.FormValue(p + fieldType)
		f.Family = FamilyOf(f.Type)
		f.TypeLabel = TypeLabel(translate, f.Type)
		cred.Type = f.Type

		switch {
		case slices.Contains(own.Remaining(creds), f.Type):
		case slices.Contains(own.Allowed, f.Type):
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
	case familyPair:
		f.readPreConnection(r, cred)
	}

	if err != nil {
		return err
	}

	f.HasErrors = f.TypeError != "" || f.Password.Error != "" || f.Pem.Error != "" ||
		f.PemCert.Error != "" || f.PemKey.Error != "" || f.PreLogin.Error != "" || f.PrePassword.Error != ""

	return nil
}

// readPreConnection takes the login and the password of a PeSIT
// pre-connection, both optional (the partner may expect an empty one): the
// login is what is posted, an empty password keeps the current one on
// edition. The module checks their length and their characters when the
// means is saved.
func (f *Form) readPreConnection(r *http.Request, cred *model.Credential) {
	f.PreLogin.Value = strings.TrimSpace(r.FormValue(f.Prefix + fieldPreLogin))
	cred.Value = f.PreLogin.Value

	if password := r.FormValue(f.Prefix + fieldPrePassword); password != "" {
		cred.Value2 = password
	}
}

// readPassword takes the password; empty, it is required at creation and
// kept on edition.
func (f *Form) readPassword(translate gui.Translator, r *http.Request, cred *model.Credential) {
	if password := r.FormValue(f.Prefix + fieldPassword); password != "" {
		cred.Value = password
	} else if !f.Edit {
		f.Password.Error = translate("auth.form.error.required")
	}
}

// readFile takes the one file of the means, uploaded or pasted. A pasted
// private key is not echoed back on a refused form; the public contents are.
// A trusted certificate given with its chain keeps the certificate alone
// (the trust bears on it, strictly), and the form says it.
func (f *Form) readFile(translate gui.Translator, r *http.Request, cred *model.Credential) error {
	content, err := FilesOrText(r, f.Prefix+fieldFile, f.Prefix+fieldPem)
	if err != nil {
		return err
	}

	if f.Type != sftp.AuthSSHPrivateKey {
		f.Pem.Value = r.FormValue(f.Prefix + fieldPem)
	}

	if content != "" && f.Type == auth.TLSTrustedCertificate {
		content = f.assemble(translate, content, &f.Pem, true)
	}

	if content != "" && f.Type != sftp.AuthSSHPrivateKey {
		// Shown again in the paste field if the form comes back refused: a
		// browser cannot show a chosen file again, and the content is public.
		f.Pem.Value = content
	}

	if content != "" {
		cred.Value = content
	} else if !f.Edit {
		f.Pem.Error = translate("auth.form.error.required")
	}

	return nil
}

// assemble puts the certificates given (one or several files, or a paste,
// in any order) in the order TLS expects: the certificate, then its
// intermediates; a root given with them is set apart and said. A trusted
// certificate keeps the certificate alone. A certificate foreign to the
// chain, or a set without its certificate, is refused on the field. A text
// that holds no certificate is returned as given: the product says why it
// refuses it.
func (f *Form) assemble(translate gui.Translator, pemText string, field *protoform.Field, leafOnly bool) string {
	certs, err := certchain.Parse(pemText)
	if err != nil {
		return pemText
	}

	chain, err := certchain.Assemble(certs)

	var unrelated *certchain.UnrelatedError

	switch {
	case errors.As(err, &unrelated):
		field.Error = translate("auth.form.error.unrelated", "name", unrelated.Subject)

		return pemText
	case err != nil:
		field.Error = translate("auth.form.error.no_leaf")

		return pemText
	}

	f.chain = chain

	if leafOnly {
		if len(certs) > 1 {
			f.Note = noteTrustedLeafOnly
		}

		return chain.LeafPEM()
	}

	if chain.Root != nil {
		f.Note, f.Root = noteRootDropped, chain.Root.Subject.CommonName
	}

	return chain.PEM()
}

// readPair takes the certificate (with its intermediates: several files, or
// a paste of several blocks, in any order) and its private key, each one
// uploaded or pasted, each one kept on edition when left empty.
func (f *Form) readPair(translate gui.Translator, r *http.Request, cred *model.Credential) error {
	cert, err := FilesOrText(r, f.Prefix+fieldCertFile, f.Prefix+fieldCertPem)
	if err != nil {
		return err
	}

	if cert != "" {
		cert = f.assemble(translate, cert, &f.PemCert, false)
	}

	key, err := FileOrText(r, f.Prefix+fieldKeyFile, f.Prefix+fieldKeyPem)
	if err != nil {
		return err
	}

	// Shown again in the paste field if the form comes back refused: what was
	// pasted, or the chain uploaded (a browser cannot show a chosen file
	// again, and the chain is public).
	f.PemCert.Value = r.FormValue(f.Prefix + fieldCertPem)
	if cert != "" {
		f.PemCert.Value = cert
	}

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

// Refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *Form) Refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "same name"):
		f.Name.Error = translate("auth.form.error.name_taken")
	case strings.Contains(msg, "already has a"):
		f.TypeError = translate("auth.form.error.unique")
	case strings.Contains(msg, "failed to validate authentication value"):
		_, reason, _ := strings.Cut(msg, "failed to validate authentication value: ")
		text, onKey := f.explain(translate, reason)

		switch f.Family {
		case familyPassword:
			f.Password.Error = text
		case familyFile:
			f.Pem.Error = text
		case familyTwoFile:
			if onKey {
				f.PemKey.Error = text
			} else {
				f.PemCert.Error = text
			}
		case familyPair:
			f.refusePreConnection(reason, text)
		default:
			f.Refused = text
		}
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// explain says a refusal of the product in the words of the user: the chain
// reaches no authority the gateway knows (when the root came with the chain,
// it is that root which is not registered: said alone, nothing else to give);
// a certificate was given in place of the private key; the key is not the one
// of the certificate; the key cannot be read. Any other cause is shown as the
// product says it. The second value says whether the cause concerns the
// private key field.
func (f *Form) explain(translate gui.Translator, reason string) (text string, onKey bool) {
	switch {
	case f.chain != nil && strings.Contains(reason, causeUnknownAuthority):
		subject, issuer := f.chain.Unknown()
		if root := f.chain.Root; root != nil && bytes.Equal(f.chain.Last().RawIssuer, root.RawSubject) {
			return translate("auth.form.error.root_unknown", "root", issuer), false
		}

		return translate("auth.form.error.issuer_unknown", "subject", subject, "issuer", issuer), false
	case strings.Contains(reason, causeKeyIsCert):
		return translate("auth.form.error.key_is_cert"), true
	case strings.Contains(reason, causeKeyMismatch):
		return translate("auth.form.error.key_mismatch"), true
	case strings.Contains(reason, causeKeyUnreadable), strings.Contains(reason, causeKeyEmpty):
		return translate("auth.form.error.key_unreadable"), true
	default:
		return translate("auth.form.error.invalid", "reason", reason), false
	}
}

// refusePreConnection puts the refusal of a PeSIT pre-connection on the
// field it names: the module refuses a login or a password on its own.
func (f *Form) refusePreConnection(reason, text string) {
	if strings.Contains(reason, "login") {
		f.PreLogin.Error = text

		return
	}

	f.PrePassword.Error = text
}

// ------------------------------------------------------------ handlers ----

// NewForm serves GET …/auth/new.
func (p *Pages) NewForm(w http.ResponseWriter, r *http.Request, own *Owner) {
	creds, err := own.Table.GetCredentials(p.DB)
	if err != nil {
		p.Logger.Errorf("failed to list the authentication means: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	p.formPage(w, r, own, newForm(p.Shell.Translator(r), own, nil, creds, time.Now()), 0)
}

// Create serves POST …/auth/new.
func (p *Pages) Create(w http.ResponseWriter, r *http.Request, own *Owner) {
	p.save(w, r, own, nil)
}

// EditForm serves GET …/auth/{cid}/edit.
func (p *Pages) EditForm(w http.ResponseWriter, r *http.Request, own *Owner) {
	cred, ok := p.credential(w, r, own)
	if !ok {
		return
	}

	p.formPage(w, r, own, newForm(p.Shell.Translator(r), own, cred, nil, time.Now()), 0)
}

// Update serves POST …/auth/{cid}/edit.
func (p *Pages) Update(w http.ResponseWriter, r *http.Request, own *Owner) {
	cred, ok := p.credential(w, r, own)
	if !ok {
		return
	}

	p.save(w, r, own, cred)
}

// formPage renders the form of a means with the tab strip of its owner.
func (p *Pages) formPage(w http.ResponseWriter, r *http.Request, own *Owner, f *Form, status int) {
	tabs, ok := p.tabs(w, own)
	if !ok {
		return
	}

	f.Tabs = tabs
	p.Shell.Render(w, p.Logger, r, own.screen(f.Title, f.Title, status, nil), formView(f))
}

// save reads, validates and stores the posted means: a new one when cred is
// nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects to the "Authentication" tab of the
// owner with a notification.
func (p *Pages) save(w http.ResponseWriter, r *http.Request, own *Owner, cred *model.Credential) {
	translate := p.Shell.Translator(r)

	if err := ParseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	creds, err := own.Table.GetCredentials(p.DB)
	if err != nil {
		p.Logger.Errorf("failed to list the authentication means: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	form := newForm(translate, own, cred, creds, time.Now())
	done := liststate.DoneUpdated

	if cred == nil {
		cred = &model.Credential{}
		own.Table.SetCredOwner(cred)
		done = liststate.DoneCreated
	}

	if readErr := form.Read(translate, r, own, cred, creds); readErr != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	if !form.HasErrors {
		var saveErr error
		if done == liststate.DoneCreated {
			saveErr = p.DB.Insert(cred).Run()
		} else {
			saveErr = p.DB.Update(cred).Run()
		}

		switch {
		case saveErr == nil:
			p.redirect(w, r, own, done, cred.Name, form.Note, form.Root)

			return
		case database.IsValidationError(saveErr):
			form.Refuse(translate, saveErr)
		default:
			p.Logger.Errorf("failed to save the authentication means %q: %v", cred.Name, saveErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	if !form.Edit {
		form.Types = TypeChoices(translate, own.Remaining(creds), form.Type, false)
	}

	p.formPage(w, r, own, form, http.StatusUnprocessableEntity)
}

// redirect sends to the "Authentication" tab of the owner (carrying the list
// state) with the result of the action on the means, and what the form did
// with the certificates given beyond the chain, when it did something.
func (p *Pages) redirect(w http.ResponseWriter, r *http.Request, own *Owner, done, name, note, root string) {
	values := url.Values{
		liststate.KindParam: {KindCred}, liststate.DoneParam: {done}, liststate.NameParam: {name},
	}

	if note != "" {
		values.Set(NoteParam, note)
	}

	if root != "" {
		values.Set(RootParam, root)
	}

	liststate.RedirectTo(w, r, own.WithRet(own.AuthURL), values)
}

// Notices turns the result of an action on a means, carried by the
// redirection, into notifications: the action done, then what the form did
// with the certificates given beyond the chain.
func Notices(translate gui.Translator, values url.Values) []gui.Notice {
	var notices []gui.Notice

	name, done := values.Get(liststate.NameParam), values.Get(liststate.DoneParam)
	if slices.Contains([]string{liststate.DoneCreated, liststate.DoneUpdated, liststate.DoneDeleted}, done) {
		notices = append(notices, gui.Notice{Kind: "ok", Title: translate("auth.toast."+done, "name", name)})
	}

	switch values.Get(NoteParam) {
	case noteRootDropped:
		notices = append(notices, gui.Notice{
			Kind: "warn", Title: translate("auth.note.root_dropped", "root", values.Get(RootParam)),
		})
	case noteTrustedLeafOnly:
		notices = append(notices, gui.Notice{Kind: "warn", Title: translate("auth.note.trusted_leaf_only")})
	}

	return notices
}

// ---------------------------------------------------------------- forms ----

// ParseForm reads the posted form, multipart (with files) or not.
func ParseForm(r *http.Request) error {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		if err := r.ParseMultipartForm(maxUpload); err != nil {
			return fmt.Errorf("failed to read the multipart form: %w", err)
		}

		return nil
	}

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to read the form: %w", err)
	}

	return nil
}

// FilesOrText is the content of the files uploaded in a field, joined in the
// order given (a certificate and its intermediates), or, without any, of the
// text field next to it where the same content can be pasted.
func FilesOrText(r *http.Request, fileField, textField string) (string, error) {
	if r.MultipartForm == nil || len(r.MultipartForm.File[fileField]) <= 1 {
		return FileOrText(r, fileField, textField)
	}

	var parts []string

	for _, header := range r.MultipartForm.File[fileField] {
		file, err := header.Open()
		if err != nil {
			return "", fmt.Errorf("failed to read the uploaded file: %w", err)
		}

		content, readErr := io.ReadAll(io.LimitReader(file, maxUpload))

		file.Close() //nolint:errcheck,gosec // read only

		if readErr != nil {
			return "", fmt.Errorf("failed to read the uploaded file: %w", readErr)
		}

		if text := normalizeText(string(content)); text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "\n"), nil
}

// FileOrText is the content of the file uploaded in a field or, without one,
// of the text field next to it where the same content can be pasted. Line
// ends are normalised and the content trimmed, so that a file written on
// any system is read the same way.
func FileOrText(r *http.Request, fileField, textField string) (string, error) {
	if r.MultipartForm != nil {
		file, _, err := r.FormFile(fileField)

		switch {
		case err == nil:
			defer file.Close() //nolint:errcheck // read only

			content, readErr := io.ReadAll(io.LimitReader(file, maxUpload))
			if readErr != nil {
				return "", fmt.Errorf("failed to read the uploaded file: %w", readErr)
			}

			if len(content) > 0 {
				return normalizeText(string(content)), nil
			}
		case !errors.Is(err, http.ErrMissingFile):
			return "", fmt.Errorf("failed to read the uploaded file: %w", err)
		}
	}

	return normalizeText(r.FormValue(textField)), nil
}

func normalizeText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}
