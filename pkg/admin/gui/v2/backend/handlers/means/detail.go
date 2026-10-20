package means

import (
	"net/http"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

// kv is a row of a key/value list.
type kv = protoform.KV

type detailData struct {
	Title       string
	Name        string
	Owner       string
	Type        string
	Tabs        []Tab
	TabsLabel   string
	BackURL     string
	EditURL     string
	DeleteURL   string
	Expiry      gui.Status
	Rows        []kv
	Public      string // the public part, when there is one
	PublicTitle string
}

// buildDetail describes a means in reading: what identifies it, never the
// secret.
func buildDetail(translate gui.Translator, own *Owner, cred *view, now time.Time) detailData {
	data := detailData{
		Title:     translate("auth.detail.title", "name", cred.Name),
		Name:      cred.Name,
		Owner:     own.Label,
		Type:      TypeLabel(translate, cred.Type),
		TabsLabel: translate(own.Texts.Tabs),
		BackURL:   own.WithRet(own.AuthURL),
		Expiry:    cred.expiry(translate, now),
	}

	if own.Rights.Write {
		data.EditURL = own.WithRet(own.CredURL(cred.ID, EditSuffix))
	}

	if own.Rights.Delete {
		data.DeleteURL = own.WithRet(own.CredURL(cred.ID, DeleteSuffix))
	}

	never := translate("auth.detail.never_shown")
	date := translate("auth.date")

	switch {
	case cred.isCert() && cred.Subject != "":
		issuer := kv{Label: translate("auth.detail.issuer"), Value: cred.Issuer, Mono: true}
		if cred.SelfSigned {
			issuer = kv{Label: translate("auth.detail.issuer"), Value: translate("auth.detail.self_signed")}
		}

		data.Rows = []kv{
			{Label: translate("auth.detail.subject"), Value: cred.Subject, Mono: true},
			issuer,
			{
				Label: translate("auth.detail.validity"),
				Value: translate("auth.detail.validity_text",
					"from", cred.NotBefore.Format(date), "to", cred.NotAfter.Format(date)),
			},
			{Label: translate("auth.detail.fingerprint"), Value: cred.Fingerprint, Mono: true},
		}

		if cred.Type == auth.TLSCertificate {
			data.Rows = append(data.Rows, kv{Label: translate("auth.detail.private_key"), Value: never})
		}

		data.Public, data.PublicTitle = cred.Public, translate("auth.detail.pem")
	case cred.Type == sftp.AuthSSHPublicKey && cred.KeyType != "":
		data.Rows = []kv{
			{Label: translate("auth.detail.key_type"), Value: cred.KeyType, Mono: true},
			{Label: translate("auth.detail.fingerprint"), Value: cred.Fingerprint, Mono: true},
		}
		data.Public, data.PublicTitle = cred.Public, translate("auth.detail.public_key")
	case cred.Type == sftp.AuthSSHPrivateKey && cred.KeyType != "":
		data.Rows = []kv{
			{Label: translate("auth.detail.key_type"), Value: cred.KeyType, Mono: true},
			{Label: translate("auth.detail.fingerprint"), Value: cred.Fingerprint, Mono: true},
			{Label: translate("auth.detail.private_key"), Value: never},
		}
	case cred.Type == pesit.PreConnectionAuth:
		data.Rows = preConnectionRows(translate, cred, never)
	case cred.Type == r66auth.AuthLegacyCertificate:
		data.Rows = []kv{{Label: translate("auth.detail.content"), Value: translate("auth.detail.legacy")}}
	default:
		data.Rows = []kv{{Label: translate("auth.detail.content"), Value: never}}
	}

	return data
}

// preConnectionRows are the rows of a PeSIT pre-connection: its login, and
// whether it holds a password; neither is required.
func preConnectionRows(translate gui.Translator, cred *view, never string) []kv {
	login := kv{Label: translate("auth.detail.pre_login"), Value: translate("auth.detail.pre_no_login")}
	if cred.Login != "" {
		login = kv{Label: translate("auth.detail.pre_login"), Value: cred.Login, Mono: true}
	}

	password := kv{Label: translate("auth.detail.pre_password"), Value: translate("auth.detail.pre_no_password")}
	if cred.HasPassword {
		password = kv{Label: translate("auth.detail.pre_password"), Value: never}
	}

	return []kv{login, password}
}

// Detail serves GET …/auth/{cid}: a means in reading.
func (p *Pages) Detail(w http.ResponseWriter, r *http.Request, own *Owner) {
	cred, ok := p.credential(w, r, own)
	if !ok {
		return
	}

	v := viewOf(cred)
	data := buildDetail(p.Shell.Translator(r), own, &v, time.Now())

	tabs, ok := p.tabs(w, own)
	if !ok {
		return
	}

	data.Tabs = tabs
	p.Shell.Render(w, p.Logger, r, own.screen(data.Title, cred.Name, 0, nil), detailView(data))
}

// -------------------------------------------------------------- delete ----

type deleteData struct {
	Title     string
	Text      string
	Name      string
	Type      string
	Owner     string
	Summary   string
	Tabs      []Tab
	TabsLabel string
	CancelURL string
	Action    string
}

func (p *Pages) deletePage(w http.ResponseWriter, r *http.Request, own *Owner, cred *model.Credential) {
	translate := p.Shell.Translator(r)
	v := viewOf(cred)

	data := deleteData{
		Title:     translate("auth.delete.title", "name", cred.Name),
		Text:      translate(own.Texts.Delete),
		Name:      cred.Name,
		Type:      TypeLabel(translate, cred.Type),
		Owner:     own.Label,
		Summary:   v.summary(translate),
		TabsLabel: translate(own.Texts.Tabs),
		CancelURL: own.WithRet(own.CredURL(cred.ID, "")),
		Action:    own.WithRet(own.CredURL(cred.ID, DeleteSuffix)),
	}

	tabs, ok := p.tabs(w, own)
	if !ok {
		return
	}

	data.Tabs = tabs
	p.Shell.Render(w, p.Logger, r, own.screen(data.Title, data.Title, 0, nil), deleteView(data))
}

// DeleteForm serves GET …/auth/{cid}/delete: the confirmation page.
func (p *Pages) DeleteForm(w http.ResponseWriter, r *http.Request, own *Owner) {
	cred, ok := p.credential(w, r, own)
	if !ok {
		return
	}

	p.deletePage(w, r, own, cred)
}

// Delete serves POST …/auth/{cid}/delete.
func (p *Pages) Delete(w http.ResponseWriter, r *http.Request, own *Owner) {
	cred, ok := p.credential(w, r, own)
	if !ok {
		return
	}

	if err := p.DB.Delete(cred).Run(); err != nil {
		p.Logger.Errorf("failed to delete the authentication means %q: %v", cred.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	p.redirect(w, r, own, liststate.DoneDeleted, cred.Name)
}
