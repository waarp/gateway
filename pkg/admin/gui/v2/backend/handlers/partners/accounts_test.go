package partners

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
)

const (
	prodPassword = "s3cret" // the password of the "prod" account of the fixtures
	prePassword  = "pa55"
)

func (s *suite) accountPath(acc *model.RemoteAccount, suffix string) string {
	return fmt.Sprintf("/webui/partners/%d/accounts/%d%s", acc.RemoteAgentID, acc.ID, suffix)
}

func (s *suite) credCount(t *testing.T, owner model.CredOwnerTable) uint64 {
	t.Helper()

	n, err := s.db.Count(&model.Credential{}).Where(owner.GetCredCond()).Run()
	require.NoError(t, err)

	return n
}

// credOf reads a means back from the database: the model decrypts its
// secret on reading.
func (s *suite) credOf(t *testing.T, id int64) *model.Credential {
	t.Helper()

	var cred model.Credential
	require.NoError(t, s.db.Get(&cred, "id=?", id).Run())

	return &cred
}

// preConnection adds to the PeSIT partner an account with a pre-connection
// (a login and a password).
func (s *suite) preConnection(t *testing.T, login string) (*model.RemoteAccount, *model.Credential) {
	t.Helper()

	acc := &model.RemoteAccount{RemoteAgentID: s.pesit.ID, Login: login}
	require.NoError(t, s.db.Insert(acc).Run())

	cred := &model.Credential{
		RemoteAccountID: acc.NullableID(), Name: "pre-connexion", Type: pesit.PreConnectionAuth,
		Value: "GWLOGIN", Value2: prePassword,
	}
	require.NoError(t, s.db.Insert(cred).Run())

	return acc, cred
}

func TestAccountsTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the SFTP partner and every right", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.sftp, "/accounts")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dpart" aria-current="page">Comptes distants <span class="w-tabs__count">2</span></a>`)
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/accounts/new")+`?ret=q%3Dpart">`)
		assert.Contains(t, body, "2 comptes")
		indexes(t, body, "<tbody>", `>prod</a>`, `>test</a>`)
		assert.Contains(t, body, `<span class="w-status w-status--warn">Aucun moyen : ne peut pas se connecter</span>`)
		assert.Contains(t, body, "<td>Mot de passe</td>")
		assert.Contains(t, body, `href="`+s.accountPath(s.prod, "")+`?ret=q%3Dpart">`)
		assert.Contains(t, body, `href="`+s.accountPath(s.prod, "/edit")+`?ret=q%3Dpart">`)
		assert.Contains(t, body, `class="is-danger" href="`+s.accountPath(s.prod, "/delete")+`?ret=q%3Dpart">`)
		assert.NotContains(t, body, prodPassword)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a sort by authentication, ascending, read only", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.path(s.sftp, "/accounts?sort=auth")).Body.String()
		indexes(t, body, "<tbody>", `>test</a>`, `>prod</a>`)
		assert.Contains(t, body, "Comptes distants, page 1 sur 1, triés par authentification")
		assert.NotContains(t, body, "w-btn--primary")
		assert.NotContains(t, body, "/edit")
	})

	t.Run("Given a search without match", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.sftp, "/accounts?q=zzz")).Body.String()
		assert.Contains(t, body, "Aucun compte ne correspond à « zzz »")
	})

	t.Run("Given a partner without any account", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.r66, "/accounts")).Body.String()
		assert.Contains(t, body, "Aucun compte distant pour l&#39;instant")
		assert.Contains(t, body, `href="`+s.path(s.r66, "/accounts/new")+`">`)
	})
}

func TestAccountDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the prod account and every right", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.accountPath(s.prod, "")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Compte « prod » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "")+`?ret=q%3Dpart">part-sftp-01</a></li>`, "the breadcrumb goes through the partner")
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dpart">Comptes distants</a></li>`)
		assert.Contains(t, body, `<a href="`+s.accountPath(s.prod, "")+`?ret=q%3Dpart" aria-current="page">Détails</a>`)
		assert.Contains(t, body, `<a href="`+s.accountPath(s.prod, "/auth")+`?ret=q%3Dpart">Authentification <span class="w-tabs__count">1</span></a>`)
		assert.Contains(t, body, "<dt>Partenaire</dt><dd>part-sftp-01 (SFTP)</dd>")
		assert.Contains(t, body, `<dt>Identifiant</dt><dd><code class="w-code">prod</code></dd>`)
		assert.Contains(t, body, "<dt>Moyens d&#39;authentification</dt><dd>Mot de passe</dd>")
		assert.Contains(t, body, `href="`+s.accountPath(s.prod, "/edit")+`?ret=q%3Dpart">`)
		assert.Contains(t, body, `class="w-icon-btn is-danger" href="`+s.accountPath(s.prod, "/delete")+`?ret=q%3Dpart"`,
			"the only entry of the menu is an icon button")
		assert.NotContains(t, body, "Voir le compte", "no link to itself")
	})

	t.Run("Given the account without any means, read only", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.accountPath(s.test, "")).Body.String()
		assert.Contains(t, body, "<dt>Moyens d&#39;authentification</dt><dd>Aucun</dd>")
		assert.NotContains(t, body, "/edit")
		assert.NotContains(t, body, "/delete")
	})

	t.Run("Given an unknown account", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.sftp, "/accounts/999")+anyRet)
		require.Equal(t, http.StatusNotFound, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Compte distant introuvable · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dpart">`)
	})

	t.Run("Given an account of another partner", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, s.path(s.r66, fmt.Sprintf("/accounts/%d", s.prod.ID))).Code)
	})
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the form of a new account of the PeSIT partner", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.path(s.pesit, "/accounts/new")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouveau compte distant · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Compte sur le partenaire « part-pesit-01 »")
		assert.Contains(t, body, `<select class="w-select" id="means_type" name="means_type" data-switch="means_type"`)
		assert.Contains(t, body, `<option value="" selected>Aucun pour l&#39;instant</option><option value="password">Mot de passe</option>`+
			`<option value="pesit_pre-connection_auth">Pré-connexion PeSIT</option></select>`)
		assert.Contains(t, body, `data-case="pesit_pre-connection_auth" data-of="means_type"`)
		assert.Contains(t, body, `name="means_pre_login"`)
		assert.Contains(t, body, `name="means_pre_password"`)
		assert.Contains(t, body, "huit caractères ASCII au plus chacun")
		assert.Contains(t, body, `aria-current="page">Comptes distants`, "under the accounts tab")
		assert.Contains(t, body, `href="`+s.path(s.pesit, "/accounts")+`?ret=q%3Dpart">`)
	})

	t.Run("Given an account posted with a pre-connection as its first means", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.pesit, "/accounts/new")+anyRet, url.Values{
			"login": {" gw-out "}, "means_name": {""}, "means_type": {pesit.PreConnectionAuth},
			"means_pre_login": {"GWLOGIN"}, "means_pre_password": {prePassword},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, s.path(s.pesit, "/accounts")+"?done=created&kind=account&name=gw-out&ret=q%3Dpart", rec.Header().Get("Location"))

		var acc model.RemoteAccount
		require.NoError(t, s.db.Get(&acc, "login=?", "gw-out").Run())
		assert.Equal(t, s.pesit.ID, acc.RemoteAgentID)

		creds, err := acc.GetCredentials(s.db)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.Equal(t, "Pré-connexion PeSIT", creds[0].Name, "an empty name takes the name of the kind")
		assert.Equal(t, pesit.PreConnectionAuth, creds[0].Type)
		assert.Equal(t, "GWLOGIN", creds[0].Value)
		assert.Equal(t, prePassword, creds[0].Value2, "stored encrypted, read decrypted")

		body := s.get(t, permWrite, rec.Header().Get("Location")).Body.String()
		assert.Contains(t, body, "Compte « gw-out » créé")
		assert.Contains(t, body, `>gw-out</a>`)
		assert.Contains(t, body, "<td>Pré-connexion PeSIT</td>")
		assert.NotContains(t, body, prePassword)
	})

	t.Run("Given an account posted without any means", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.r66, "/accounts/new"), url.Values{"login": {"alone"}, "means_type": {""}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		var acc model.RemoteAccount
		require.NoError(t, s.db.Get(&acc, "login=?", "alone").Run())
		assert.EqualValues(t, 0, s.credCount(t, &acc))
	})

	t.Run("Given forms the screen or the model refuses", func(t *testing.T) {
		t.Parallel()

		before, err := s.db.Count(&model.RemoteAccount{}).Where("remote_agent_id=?", s.sftp.ID).Run()
		require.NoError(t, err)

		rec := s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{"login": {""}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez l&#39;identifiant du compte.")

		rec = s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{"login": {"prod"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre compte de ce partenaire porte déjà cet identifiant.")

		// The first means is refused: the account is not created either.
		rec = s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{
			"login": {"half"}, "means_type": {"password"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez le contenu du moyen d&#39;authentification.")

		rec = s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{
			"login": {"half"}, "means_type": {pesit.PreConnectionAuth}, "means_pre_login": {"GW"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Choisissez un type admis par le protocole.", "no pre-connection over SFTP")

		after, err := s.db.Count(&model.RemoteAccount{}).Where("remote_agent_id=?", s.sftp.ID).Run()
		require.NoError(t, err)
		assert.Equal(t, before, after, "nothing was saved")
	})

	t.Run("Given a pre-connection the module refuses", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.pesit, "/accounts/new"), url.Values{
			"login": {"long"}, "means_type": {pesit.PreConnectionAuth}, "means_pre_login": {"NINECHARS"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le contenu n&#39;a pas été accepté : pesit logins cannot be longer than 8 characters")
		assert.Contains(t, rec.Body.String(), `value="NINECHARS"`, "the login is kept")
		assert.Contains(t, rec.Body.String(), `<option value="pesit_pre-connection_auth" selected>`)

		rec = s.post(t, permWrite, s.path(s.pesit, "/accounts/new"), url.Values{
			"login": {"long"}, "means_type": {pesit.PreConnectionAuth}, "means_pre_password": {"ninechars"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "pesit passwords cannot be longer than 8 characters")
		assert.NotContains(t, rec.Body.String(), "ninechars", "a refused password is not echoed")

		n, err := s.db.Count(&model.RemoteAccount{}).Where("login=?", "long").Run()
		require.NoError(t, err)
		assert.Zero(t, n, "nothing was saved")
	})
}

func TestEditAccount(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permWrite, s.accountPath(s.prod, "/edit")+anyRet)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "<title>Modifier « prod » · gw-test · Waarp Gateway</title>")
	assert.Contains(t, body, `value="prod"`)
	assert.NotContains(t, body, "means_type", "no first means on edition")
	assert.Contains(t, body, `aria-current="page">Détails</a>`, "under the tabs of the account")
	assert.Contains(t, body, `href="`+s.accountPath(s.prod, "")+`?ret=q%3Dpart">`, "cancel goes back to the account")

	rec = s.post(t, permWrite, s.accountPath(s.prod, "/edit")+anyRet, url.Values{"login": {"prod-2"}})
	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
	assert.Equal(t, s.accountPath(s.prod, "")+"?done=updated&kind=account&name=prod-2&ret=q%3Dpart", rec.Header().Get("Location"))

	var acc model.RemoteAccount
	require.NoError(t, s.db.Get(&acc, "id=?", s.prod.ID).Run())
	assert.Equal(t, "prod-2", acc.Login)
	assert.EqualValues(t, 1, s.credCount(t, &acc), "the means are kept")

	body = s.get(t, permWrite, rec.Header().Get("Location")).Body.String()
	assert.Contains(t, body, "Compte « prod-2 » enregistré")

	rec = s.post(t, permWrite, s.accountPath(s.prod, "/edit"), url.Values{"login": {"test"}})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Un autre compte de ce partenaire porte déjà cet identifiant.")
}

func TestDeleteAccount(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given an account used by a running transfer", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, s.db.Insert(&model.Transfer{
			RuleID: s.send.ID, ClientID: sql.NullInt64{Int64: s.cliSFTP.ID, Valid: true},
			RemoteAccountID: s.test.NullableID(), SrcFilename: "file",
		}).Run())

		rec := s.get(t, permAll, s.accountPath(s.test, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<strong>Ce compte ne peut pas être supprimé.</strong>Des transferts en cours l&#39;utilisent.")
		assert.Contains(t, body, `href="/webui/transfer_monitoring">`)
		assert.NotContains(t, body, `<form class="sheet-form-actions"`, "no deletion offered")

		rec = s.post(t, permAll, s.accountPath(s.test, "/delete"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("Given an account used by a folder watcher", func(t *testing.T) {
		t.Parallel()

		watched := &model.RemoteAccount{RemoteAgentID: s.sftp.ID, Login: "watched"}
		require.NoError(t, s.db.Insert(watched).Run())
		require.NoError(t, s.db.Insert(&model.FileWatcher{
			Flow: "flux-compte", RemoteAccount: *watched, Client: *s.cliSFTP, Rule: *s.receive,
		}).Run())

		rec := s.get(t, permAll, s.accountPath(watched, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "<strong>Ce compte ne peut pas être supprimé.</strong>Une surveillance de dossiers l&#39;utilise.")
		assert.NotContains(t, rec.Body.String(), `<form class="sheet-form-actions"`)

		rec = s.post(t, permAll, s.accountPath(watched, "/delete"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("Given the prod account", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.accountPath(s.prod, "/delete")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Supprimer le compte « prod » ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "les transferts vers « part-sftp-01 » qui le nomment")
		assert.Contains(t, body, "<dt>Moyens d&#39;authentification</dt><dd>Mot de passe</dd>")
		assert.NotContains(t, body, prodPassword)

		rec = s.post(t, permAll, s.accountPath(s.prod, "/delete")+anyRet, nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, s.path(s.sftp, "/accounts")+"?done=deleted&kind=account&name=prod&ret=q%3Dpart", rec.Header().Get("Location"))

		n, err := s.db.Count(&model.Credential{}).Where("remote_account_id=?", s.prod.ID).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 0, n, "the means go with the account")

		body = s.get(t, permAll, rec.Header().Get("Location")).Body.String()
		assert.Contains(t, body, "Compte « prod » supprimé")
		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, s.accountPath(s.prod, "")).Code)
	})
}

func TestAccountAuthTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	acc, cred := s.preConnection(t, "gw-out")

	t.Run("Given the password of the prod account", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.accountPath(s.prod, "/auth")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Ce que la passerelle présente au partenaire")
		assert.Contains(t, body, `aria-current="page">Authentification <span class="w-tabs__count">1</span></a>`)
		assert.Contains(t, body, `href="`+s.accountPath(s.prod, "/auth/new")+anyRet+`"`, "a private key can still be added")
		assert.Contains(t, body, "Défini, non affiché")
		assert.NotContains(t, body, prodPassword)

		body = s.get(t, permWrite, s.accountPath(s.prod, "/auth/new")).Body.String()
		assert.Contains(t, body, "Moyen d&#39;authentification pour le compte « prod » du partenaire « part-sftp-01 »")
		assert.Contains(t, body, `<option value="ssh_private_key"`)
		assert.NotContains(t, body, `<option value="password"`, "one password per account")
		assert.NotContains(t, body, `<option value="pesit_pre-connection_auth"`, "not over SFTP")
	})

	t.Run("Given the pre-connection of the PeSIT account", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, s.accountPath(acc, "/auth")).Body.String()
		assert.Contains(t, body, ">pre-connexion</a>")
		assert.Contains(t, body, "<td>Pré-connexion PeSIT</td>")
		assert.Contains(t, body, "Identifiant « GWLOGIN » · mot de passe défini, non affiché")
		assert.NotContains(t, body, prePassword)

		body = s.get(t, permWrite, s.accountPath(acc, "/auth/new")).Body.String()
		assert.Contains(t, body, `<option value="password"`)
		assert.NotContains(t, body, `<option value="pesit_pre-connection_auth"`, "one pre-connection per account")

		body = s.get(t, permAll, s.accountPath(acc, fmt.Sprintf("/auth/%d", cred.ID))).Body.String()
		assert.Contains(t, body, `<dt>Identifiant de pré-connexion</dt><dd><code class="w-code">GWLOGIN</code></dd>`)
		assert.Contains(t, body, "<dt>Mot de passe de pré-connexion</dt><dd>Définie, jamais affichée</dd>")
		assert.NotContains(t, body, prePassword)
	})

	t.Run("Given the pre-connection edited", func(t *testing.T) {
		t.Parallel()

		// Its own account: the other subtests read the first one.
		acc, cred := s.preConnection(t, "gw-edit")

		rec := s.get(t, permWrite, s.accountPath(acc, fmt.Sprintf("/auth/%d/edit", cred.ID)))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `value="GWLOGIN"`)
		assert.Contains(t, body, "mot de passe défini, non affiché · laissez vide pour ne pas en changer")
		assert.NotContains(t, body, `data-case=`, "only the block of the kind")
		assert.NotContains(t, body, prePassword)

		rec = s.post(t, permWrite, s.accountPath(acc, fmt.Sprintf("/auth/%d/edit", cred.ID)), url.Values{
			"name": {"pre-2"}, "pre_login": {"GW2"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		after := s.credOf(t, cred.ID)
		assert.Equal(t, "pre-2", after.Name)
		assert.Equal(t, "GW2", after.Value, "the login is what is posted")
		assert.Equal(t, prePassword, after.Value2, "an empty password keeps the current one")

		rec = s.post(t, permWrite, s.accountPath(acc, fmt.Sprintf("/auth/%d/edit", cred.ID)), url.Values{
			"name": {"pre-2"}, "pre_login": {""}, "pre_password": {"new1"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		after = s.credOf(t, cred.ID)
		assert.Empty(t, after.Value, "an empty login is allowed")
		assert.Equal(t, "new1", after.Value2)

		body = s.get(t, permWrite, s.accountPath(acc, "/auth")).Body.String()
		assert.Contains(t, body, "Sans identifiant · mot de passe défini, non affiché")
	})
}

// No secret value ever reaches a page: not the password of an account, not
// the password of a pre-connection, whatever the page of the sheet.
func TestNoSecretInAccountPages(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	acc, cred := s.preConnection(t, "gw-out")

	var prodPwd model.Credential
	require.NoError(t, s.db.Get(&prodPwd, "remote_account_id=?", s.prod.ID).Run())

	secrets := map[string]string{"the password": prodPassword, "the pre-connection password": prePassword}

	for _, path := range []string{
		s.path(s.sftp, "/accounts"), s.accountPath(s.prod, ""), s.accountPath(s.prod, "/edit"), s.accountPath(s.prod, "/delete"),
		s.accountPath(s.prod, "/auth"), s.accountPath(s.prod, fmt.Sprintf("/auth/%d", prodPwd.ID)),
		s.accountPath(s.prod, fmt.Sprintf("/auth/%d/edit", prodPwd.ID)), s.accountPath(s.prod, fmt.Sprintf("/auth/%d/delete", prodPwd.ID)),
		s.path(s.pesit, "/accounts"), s.accountPath(acc, ""), s.accountPath(acc, "/delete"),
		s.accountPath(acc, "/auth"), s.accountPath(acc, fmt.Sprintf("/auth/%d", cred.ID)),
		s.accountPath(acc, fmt.Sprintf("/auth/%d/edit", cred.ID)), s.accountPath(acc, fmt.Sprintf("/auth/%d/delete", cred.ID)),
	} {
		rec := s.get(t, permAll, path)
		require.Equalf(t, http.StatusOK, rec.Code, "GET %s", path)

		for what, secret := range secrets {
			assert.NotContainsf(t, rec.Body.String(), secret, "GET %s shows %s", path, what)
		}
	}
}
