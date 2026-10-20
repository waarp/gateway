package means

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
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

	// A PEM file weighs a few kilobytes: the limit of an upload.
	maxUpload = 4 << 20
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
	}
}

// certText summarises a certificate in place: its subject and its validity.
func certText(translate gui.Translator, v *view, now time.Time) string {
	if v.Subject == "" {
		return translate("auth.content.set")
	}

	return v.Subject + " · " + v.expiry(translate, now).Label
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
func (f *Form) readPassword(translate gui.Translator, r *http.Request, cred *model.Credential) {
	if password := r.FormValue(f.Prefix + fieldPassword); password != "" {
		cred.Value = password
	} else if !f.Edit {
		f.Password.Error = translate("auth.form.error.required")
	}
}

// readFile takes the one file of the means, uploaded or pasted. A pasted
// private key is not echoed back on a refused form; the public contents are.
func (f *Form) readFile(translate gui.Translator, r *http.Request, cred *model.Credential) error {
	content, err := FileOrText(r, f.Prefix+fieldFile, f.Prefix+fieldPem)
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
func (f *Form) readPair(translate gui.Translator, r *http.Request, cred *model.Credential) error {
	cert, err := FileOrText(r, f.Prefix+fieldCertFile, f.Prefix+fieldCertPem)
	if err != nil {
		return err
	}

	key, err := FileOrText(r, f.Prefix+fieldKeyFile, f.Prefix+fieldKeyPem)
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
			p.redirect(w, r, own, done, cred.Name)

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
// state) with the result of the action on the means.
func (p *Pages) redirect(w http.ResponseWriter, r *http.Request, own *Owner, done, name string) {
	liststate.RedirectTo(w, r, own.WithRet(own.AuthURL), url.Values{
		liststate.KindParam: {KindCred}, liststate.DoneParam: {done}, liststate.NameParam: {name},
	})
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
