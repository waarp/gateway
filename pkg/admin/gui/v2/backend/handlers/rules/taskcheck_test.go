package rules

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The typed parameters are checked by the form before the product, with a
// message under the field: what the form refuses, the product refuses too,
// but an address with "?" after "@", which the mail standard admits and no
// host name does (the form is stricter there, on purpose). A value stored
// and left as it is is not judged again.
func TestTaskChecks(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	refused := func(form url.Values, field, message string) {
		t.Helper()

		rec := s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), form)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "%s refused", field)
		assert.Contains(t, rec.Body.String(), `<span class="w-field__error" id="`+field+`-h">`, "%s: the message under the field", field)
		assert.Contains(t, rec.Body.String(), message, field)
	}

	refused(url.Values{
		"type": {tasks.Email}, "email.sender": {sender}, "email.template": {template}, "email.recipients": {"banque@banque?fr"},
	}, "email-recipients", "Indiquez des adresses e-mail séparées par des virgules")
	assert.NoError(t, validate(s.db, tasks.Email, map[string]string{
		"sender": sender, "template": template, "recipients": "banque@banque?fr",
	}), "the product admits it: the form is stricter")

	refused(url.Values{"type": {tasks.Chmod}, "chmod.perms": {"999"}}, "chmod-perms", "Indiquez des droits en octal")
	assert.Error(t, validate(s.db, tasks.Chmod, map[string]string{"perms": "999"}), "the product refuses it too")

	refused(url.Values{"type": {tasks.RemoteDelete}, "remotedelete.timeout": {"30"}}, "remotedelete-timeout",
		"Indiquez une durée, par exemple 30s, 5m ou 1h30m.")
	assert.Error(t, validate(s.db, tasks.RemoteDelete, map[string]string{"timeout": "30"}), "the product refuses it too")

	refused(url.Values{"type": {tasks.Icap}, "icap.uploadURL": {"icap://"}}, "icap-uploadURL",
		"Indiquez l&#39;adresse du serveur ICAP")

	for _, address := range []string{"icap://antivirus.exemple.fr:1344/avscan", "127.0.0.1:1344/avscan"} {
		rec := s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{"type": {tasks.Icap}, "icap.uploadURL": {address}})
		assert.Equal(t, http.StatusSeeOther, rec.Code, "%s accepted, as the gateway reads it", address)
	}

	body := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type=EMAIL").Body.String()
	assert.Contains(t, body, `aria-describedby="email-recipients-h" data-check="Indiquez des adresses e-mail séparées par des virgules,`,
		"the page checks it as soon as the user leaves the field")
	assert.Contains(t, body, `multiple spellcheck="false" type="email" data-hint="`, "with the rule of the browsers")
	assert.Contains(t, body, `src="/webui/static/gui/check.js"`)

	// A stored address the form would refuse, left as it is.
	rule := &model.Rule{Name: "adresse-api", IsSend: true, Path: "adresse-api"}
	require.NoError(t, s.db.Insert(rule).Run())
	require.NoError(t, s.db.Insert(&model.Task{RuleID: rule.ID, Chain: model.ChainPost, Type: tasks.Email, Args: map[string]string{
		"sender": sender, "template": template, "recipients": "banque@banque?fr",
	}}).Run())

	action, values := postedForm(t, s.get(t, permWrite, path(rule, "/tasks/post/0/edit")).Body.String())
	require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, action, values).Code, "saved as it was")
	assert.Equal(t, "banque@banque?fr", s.chain(t, rule, model.ChainPost)[0].Args["recipients"])
}
