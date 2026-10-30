package rules

import (
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const (
	sender   = "alertes@passerelle.example"
	template = "fin-de-transfert"
)

// taskFixtures gives "commandes" a chain after the transfer (copy, email,
// move) and a SETINFO task on error; the email names an SMTP credential and a
// template, which it returns.
func (s *suite) taskFixtures(t *testing.T) *model.SMTPCredential {
	t.Helper()

	cred := &model.SMTPCredential{EmailAddress: sender, ServerAddress: types.Addr("smtp.example.org", 587)}
	require.NoError(t, s.db.Insert(cred).Run())
	require.NoError(t, s.db.Insert(&model.EmailTemplate{Name: template, Subject: "Fin", Body: "Fichier reçu."}).Run())

	for _, task := range []*model.Task{
		{RuleID: s.orders.ID, Chain: model.ChainPost, Rank: 0, Type: tasks.Copy, Args: map[string]string{"path": "/archives"}},
		{RuleID: s.orders.ID, Chain: model.ChainPost, Rank: 1, Type: tasks.Email, Args: map[string]string{
			"sender": sender, "recipients": "exploitation@example.org", "template": template,
		}},
		{RuleID: s.orders.ID, Chain: model.ChainPost, Rank: 2, Type: tasks.Move, Args: map[string]string{"path": "/sortie"}},
		{RuleID: s.orders.ID, Chain: model.ChainError, Rank: 0, Type: tasks.SetInfo, Args: map[string]string{"lot": "A", "vide": ""}},
	} {
		require.NoError(t, s.db.Insert(task).Run())
	}

	return cred
}

// chain reads the types of a chain of a rule, by rank, with their arguments.
func (s *suite) chain(t *testing.T, rule *model.Rule, chain model.Chain) []*model.Task {
	t.Helper()

	var list model.Tasks
	require.NoError(t, s.db.Select(&list).Where("rule_id=? AND chain=?", rule.ID, chain).Run())

	out := make([]*model.Task, len(list))
	for _, task := range list {
		require.Less(t, int(task.Rank), len(list), "the ranks follow each other")
		out[task.Rank] = task
	}

	return out
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

// at is the address of a change to a task, with the version of its chain
// as the tab shows it now.
func (s *suite) at(t *testing.T, rule *model.Rule, chain model.Chain, slug string, rank int, suffix string) string {
	t.Helper()

	return withVersion(path(rule, "/tasks/"+slug+"/"+strconv.Itoa(rank)+suffix), versionOf(s.chain(t, rule, chain)))
}

func typesOf(list []*model.Task) []string {
	out := make([]string, 0, len(list))
	for _, task := range list {
		out = append(out, task.Type)
	}

	return out
}

func TestTasksTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	t.Run("Given a reader", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permRead, path(s.orders, "/tasks"))
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, `href="/webui/rules/`+itoa64(s.orders.ID)+`/tasks" aria-current="page">Tâches`)
		indexes(t, body, "Avant le transfert", "Aucune tâche.", "Après le transfert", "3 tâches",
			`<details class="rules-taskfold"><summary>`, `<strong class="rules-taskname">COPY</strong>`, "Dossier de destination",
			"/archives", `title="Copier le fichier : `,
			`<strong class="rules-taskname">EMAIL</strong>`, "Expéditeur", sender, "Envoyer un e-mail",
			`<strong class="rules-taskname">MOVE</strong>`, "/sortie", "Déplacer le fichier",
			"En cas d&#39;erreur", "1 tâche", "lot", ">A<", "vide")
		assert.Contains(t, body, `aria-label="Explication : COPY"`, "what a task does, in its tooltip")
		assert.NotContains(t, body, `<details class="rules-taskfold" open`, "the tasks folded on arrival")
		assert.Contains(t, body, `<div class="w-grid w-grid--3 rules-columns">`, "the chains side by side")
		assert.Contains(t, body, `data-key="rule-chain-post" open>`, "foldable, their state remembered")
		assert.NotContains(t, body, "Ajouter une tâche", "no addition without the write right")
		assert.NotContains(t, body, "Modifier la tâche", "nor a menu")
		assert.NotContains(t, body, "data-sortable=", "nor an order to change")
		assert.NotContains(t, body, "Introuvable", "every named object exists")
	})

	t.Run("Given a writer", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.orders, "/tasks")).Body.String()
		assert.Equal(t, 3, strings.Count(body, "Ajouter une tâche<span"), "one addition a chain")
		assert.Contains(t, body, `<span class="visually-hidden"> après le transfert</span>`, "which says its chain")
		assert.Contains(t, body, path(s.orders, "/tasks/post/new"))
		assert.Contains(t, body, path(s.orders, "/tasks/post/1/edit"))
		version := versionOf(s.chain(t, s.orders, model.ChainPost))
		assert.Contains(t, body, `action="`+path(s.orders, "/tasks/post/1/up")+`?v=`+version+`"`, "with the chain shown")
		assert.Contains(t, body, `action="`+path(s.orders, "/tasks/post/1/down")+`?v=`+version+`"`)
		assert.NotContains(t, body, path(s.orders, "/tasks/post/0/up"), "the first one does not move up")
		assert.NotContains(t, body, path(s.orders, "/tasks/post/2/down"), "the last one does not move down")
		assert.Contains(t, body, path(s.orders, "/tasks/post/2/delete"), "the write right removes a task, as in the REST API")
		assert.NotContains(t, body, "Modifier la règle", "the edition concerns the Details tab")

		order := `id="order-post" method="post" action="` + path(s.orders, "/tasks/post/order") + `?v=` + version + `"`
		assert.Contains(t, body, order, "the chain is ordered again with the state shown")
		assert.Contains(t, body, `<ol class="w-sortable rules-tasklist" aria-label="Après le transfert" data-sortable="order-post">`)
		assert.Contains(t, body, `<input type="hidden" name="rank" value="2" form="order-post">`)
		assert.Equal(t, 3, strings.Count(body, "data-sortable-grip"), "a grip a task of the chain of three")
		assert.NotContains(t, body, `id="order-error"`, "a chain of one task has no order to change")
	})

	t.Run("Given a rule that does not exist", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusNotFound, s.get(t, permRead, "/webui/rules/9999/tasks").Code)
	})
}

// The address of the former tasks page leads to the Tasks tab of its rule,
// or to the list without a valid rule.
func TestFormerTasksPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)

	for target, location := range map[string]string{
		"/webui/tasks?ruleID=12": "/webui/rules/12/tasks",
		"/webui/tasks":           "/webui/rules",
		"/webui/tasks?ruleID=x":  "/webui/rules",
		"/webui/tasks?ruleID=-3": "/webui/rules",
	} {
		rec := s.get(t, permRead, target)
		assert.Equal(t, http.StatusMovedPermanently, rec.Code, target)
		assert.Equal(t, location, rec.Header().Get("Location"), target)
	}
}

func TestTasksTabMissingObject(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	cred := s.taskFixtures(t)
	require.NoError(t, s.db.Delete(cred).Run())

	body := s.get(t, permRead, path(s.orders, "/tasks")).Body.String()
	indexes(t, body, `<strong class="rules-taskname">EMAIL</strong>`, `<span class="w-status w-status--warn">Introuvable</span>`,
		sender, `<span class="w-status w-status--warn">Introuvable</span>`, `<a href="/webui/smtp">Identifiants SMTP</a>`, template)
	assert.Equal(t, 2, strings.Count(body, "Introuvable"), "said on the folded task and at its parameter; the template is still there")

	screens := map[string]bool{}
	for _, section := range common.Screens(model.PermAll) {
		for _, screen := range section.Screens {
			screens[screen.Key] = true
		}
	}

	for _, key := range refScreens {
		assert.True(t, screens[key], "the screen %q of the screen map lists a kind of object", key)
	}

	body = s.get(t, permWrite, path(s.orders, "/tasks/post/1/edit")).Body.String()
	assert.Contains(t, body, `<option value="`+sender+`" selected>`+sender+` (introuvable)</option>`,
		"the form keeps the value, marked")
}

func TestTaskAdd(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	t.Run("Given the page of a new task", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?ret=page%3D2")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, "Nouvelle tâche (avant le transfert)")
		assert.Contains(t, body, `method="post" action="`+path(s.orders, "/tasks/pre/new")+`?ret=page%3D2"`, "one page")
		assert.Contains(t, body, `<input type="hidden" name="ret" value="page=2">`, "the list state goes along")
		assert.Contains(t, body, `formmethod="get" formaction="`+path(s.orders, "/tasks/pre/new")+`" formnovalidate>Continuer`,
			"without JavaScript, the page asked again with the type")
		indexes(t, body, `data-combobox="`, `data-switch="type" data-code-first>`, `<optgroup label="Opérations de fichiers">`,
			`<option value="CHMOD"`, `<option value="COPY" data-label="Copier le fichier" data-code="COPY" data-text="Crée une copie du fichier de transfert`,
			`>COPY · Copier le fichier</option>`, `<option value="COPYRENAME"`, `<option value="DELETE"`, `<option value="MOVE"`,
			`<option value="MOVERENAME"`, `<option value="REMOTEDELETE"`, `<optgroup label="Exécutables">`, `<optgroup label="Réseau">`,
			`<option value="EMAIL"`, `<option value="ICAP"`, `<option value="SENDMESSAGE" data-label=`, `<optgroup label="Cryptographie">`)
		assert.Contains(t, body, `src="/webui/static/gui/combobox.js"`, "the list to type in")

		for i := range taskSpecs {
			typ := html.EscapeString(taskSpecs[i].Type)
			assert.Contains(t, body, `<fieldset class="w-fieldset w-fieldset--wide rules-case" data-case="`+typ+`" data-of="type" hidden disabled>`,
				"the cards of every type, shown once it is chosen")
		}

		assert.Contains(t, body, `data-case="TRANSFER PREREGISTER" data-of="type" hidden disabled>`,
			"one card of the transfer information for both types")
	})

	t.Run("Given a type the screen does not know", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type=INCONNU")
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Choisissez un type.")

		assert.Equal(t, http.StatusNotFound, s.get(t, permWrite, path(s.orders, "/tasks/ailleurs/new")).Code,
			"nor a chain")
	})

	t.Run("Given the parameters of TRANSFER", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type=TRANSFER").Body.String()
		assert.Contains(t, body, `<option value="TRANSFER" selected data-label="Lancer un transfert"`)
		indexes(t, body, `rules-typeinfo" data-case="TRANSFER" data-of="type">`, `<code class="w-code">TRANSFER</code>`,
			`<strong>Lancer un transfert</strong>`) // its name first under the list
		assert.Contains(t, body, `data-case="TRANSFER" data-of="type">`, "its cards shown")
		indexes(t, body, `<details class="w-card w-card--fold" open><summary>`, "<h2 class=\"w-card__title\">Fichier</h2>",
			"Nom à la destination", "<h2 class=\"w-card__title\">Échange</h2>", `name="transfer.sens" value="send" required data-switch="transfer-sens"`,
			`data-case="send" data-of="transfer-sens"`, `name="transfer.rule_send"`, "Choisir une règle d&#39;envoi",
			`<option value="releves">releves</option>`,
			`data-case="receive" data-of="transfer-sens"`, `name="transfer.rule_receive"`, `<option value="commandes">commandes</option>`,
			`name="transfer.account"`, `<optgroup label="banque-demo"><option value="login=prod&amp;partner=banque-demo">prod</option>`,
			`name="transfer.using"`, "Le client du protocole du partenaire", "cli-sftp · SFTP",
			"<h2 class=\"w-card__title\">Informations de transfert</h2>", `name="info_copy"`, `data-infos`,
			"<h2 class=\"w-card__title\">Tentatives en cas d&#39;erreur</h2>", "Vides : celles du client.",
			`name="transfer.nbOfAttempts" value="" type="number" inputmode="numeric" step="1" min="0" max="127"`, "Premier délai (s)",
			"<h2 class=\"w-card__title\">Exécution</h2>", `name="transfer.synchronous" value="" checked data-switch="transfer-synchronous"`,
			`name="transfer.synchronous" value="true" data-switch="transfer-synchronous"`,
			`data-case="true" data-of="transfer-synchronous"`, `name="transfer.timeout"`)
		assert.NotContains(t, body, `name="transfer.to"`, "the direction gives the partner a side")
		assert.Contains(t, body, `data-case="TRANSFER PREREGISTER" data-of="type">`, "the card of the information shown")

		body = s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type=DELETE").Body.String()
		assert.Contains(t, body, `data-case="DELETE" data-of="type"><legend class="visually-hidden">Supprimer le fichier</legend>`+
			`<div class="w-stack"><div class="w-card"><fieldset class="w-fieldset w-fieldset--wide"><p class="rules-none w-muted">`+
			`Cette tâche n&#39;a aucun paramètre.`)

		body = s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type=ARCHIVE").Body.String()
		assert.Contains(t, body, "("+strings.Join(tasks.ArchiveExtensions.Keys(), ", ")+")", "the formats of the product")
	})

	t.Run("Given the choices of the types", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.orders, "/tasks/pre/new")+"?type=ICAP").Body.String()
		indexes(t, body, `name="icap.onError" value="" checked data-switch="icap-onError"`, "Ne rien faire",
			`name="icap.onError" value="delete"`, `name="icap.onError" value="move"`,
			`data-case="move" data-of="icap-onError"`, `name="icap.onErrorMovePath"`)
		assert.Contains(t, body, `<p class="rules-fixed"><code class="w-code">PGP</code> <span class="w-muted">seule valeur possible</span></p>`+
			`<input type="hidden" name="encrypt_sign.method" value="PGP">`, "the only method possible, said and sent")
		assert.Contains(t, body, `name="preregister.isSend" value="true" required>`, "two values as joined buttons")
		assert.Contains(t, body, `<select class="w-select" id="transcode-fromCharset" name="transcode.fromCharset" required`,
			"more values, a list")
	})

	t.Run("Given a task added after the others", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "ajout", IsSend: true, Path: "ajout"}
		require.NoError(t, s.db.Insert(rule).Run())

		rec := s.post(t, permWrite, path(rule, "/tasks/post/new")+"?ret=page%3D2", url.Values{
			"type": {tasks.Copy}, "copy.path": {" /archives "},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		location := rec.Header().Get("Location")
		assert.Contains(t, location, path(rule, "/tasks"))
		assert.Contains(t, location, "done=task_added")
		assert.Contains(t, location, "ret=page%3D2")

		rec = s.post(t, permWrite, path(rule, "/tasks/post/new"), url.Values{
			"type": {tasks.RemoteDelete}, "remotedelete.recursive": {"true"}, "remotedelete.timeout": {""},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		rec = s.post(t, permWrite, path(rule, "/tasks/post/new"), url.Values{
			"type": {tasks.RemoteDelete}, "remotedelete.recursive": {"on"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		got := s.chain(t, rule, model.ChainPost)
		assert.Equal(t, []string{tasks.Copy, tasks.RemoteDelete, tasks.RemoteDelete}, typesOf(got))
		assert.Equal(t, map[string]string{"path": "/archives"}, map[string]string(got[0].Args))
		assert.Equal(t, map[string]string{"recursive": "true"}, map[string]string(got[1].Args),
			"a ticked box as true, an empty field left out")
		assert.Empty(t, got[2].Args, "a box is ticked by true alone")

		body := s.get(t, permRead, path(rule, "/tasks")+"?done=task_added&name=Copier+le+fichier").Body.String()
		assert.Contains(t, body, "Tâche ajoutée : Copier le fichier")
	})

	t.Run("Given the tasks the former page could not save", func(t *testing.T) {
		t.Parallel()

		keys := newSuite(t)
		keys.fixtures(t)
		keys.keyFixtures(t)

		for _, form := range []url.Values{
			{"type": {tasks.Encrypt}, "encrypt.method": {"AES-CFB"}, "encrypt.keyName": {"aes"}},
			{"type": {tasks.Sign}, "sign.method": {"HMAC-SHA256"}, "sign.keyName": {"hmac"}},
			{"type": {tasks.SendMessage}, "sendmessage.partner": {"banque-demo"}, "sendmessage.account": {"prod"}, "sendmessage.client": {"cli-sftp"}},
		} {
			rec := keys.post(t, permWrite, path(keys.orders, "/tasks/post/new"), form)
			assert.Equal(t, http.StatusSeeOther, rec.Code, "%s is saved", form.Get("type"))
		}

		got := keys.chain(t, keys.orders, model.ChainPost)
		assert.Equal(t, []string{tasks.Encrypt, tasks.Sign, tasks.SendMessage}, typesOf(got))
		assert.Equal(t, map[string]string{"method": "AES-CFB", "keyName": "aes"}, map[string]string(got[0].Args))
		assert.Equal(t, map[string]string{"partner": "banque-demo", "account": "prod", "client": "cli-sftp"},
			map[string]string(got[2].Args))
	})

	t.Run("Given parameters the form refuses", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{"type": {tasks.Copy}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "La tâche n&#39;a pas été enregistrée.")
		assert.Contains(t, body, "Ce champ est obligatoire.")

		body = s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{
			"type": {tasks.Archive}, "archive.files": {"*.csv"}, "archive.compressionLevel": {"12"}, "archive.outputPath": {"/a.zip"},
		}).Body.String()
		assert.Contains(t, body, "Entre 0 et 9.")
		assert.Contains(t, body, `value="*.csv"`, "the typed values stay")

		body = s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{
			"type": {tasks.Transfer}, "transfer.file": {"f"}, "transfer.sens": {"send"}, "transfer.rule_send": {"releves"},
			"transfer.account": {accountValue("banque-demo", "prod")}, "info_name": {"lot", ""}, "info_value": {"A", "orpheline"},
			"transfer.retryIncrementFactor": {"deux"}, "transfer.firstRetryDelay": {"trente"}, "transfer.synchronous": {"true"},
		}).Body.String()
		assert.Contains(t, body, "Indiquez le nom de l&#39;information 2.")
		assert.Contains(t, body, "Indiquez un nombre.")
		assert.Contains(t, body, "Indiquez un nombre de secondes.")
		assert.Contains(t, body, `name="transfer.synchronous" value="true" checked`, "the mode chosen stays chosen")
		assert.Contains(t, body, `<option value="login=prod&amp;partner=banque-demo" selected>prod</option>`, "the account stays chosen")
		assert.Contains(t, body, `name="info_value" value="orpheline"`, "the rows stay as typed")

		body = s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{
			"type": {tasks.Transfer}, "transfer.file": {"f"},
		}).Body.String()
		assert.Contains(t, body, "Choisissez un sens.")

		body = s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{
			"type": {tasks.SetInfo}, "setinfo.pairs": {"lot=A\nsans signe"},
		}).Body.String()
		assert.Contains(t, body, "Une ligne par information, sous la forme clé=valeur.")

		assert.Empty(t, s.chain(t, s.orders, model.ChainPre), "nothing saved")
	})

	t.Run("Given parameters the product refuses", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, path(s.orders, "/tasks/pre/new"), url.Values{
			"type": {tasks.Transcode}, "transcode.fromCharset": {"klingon"}, "transcode.toCharset": {"UTF-8"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "La passerelle refuse la tâche")
		assert.Contains(t, body, "klingon (introuvable)", "the refused value stays offered")
		assert.Empty(t, s.chain(t, s.orders, model.ChainPre), "nothing saved")
	})
}

func TestTaskEdit(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	t.Run("Given a task", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, path(s.orders, "/tasks/post/1/edit"))
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, "Modifier la tâche n° 2 (après le transfert)")
		assert.Contains(t, body, "Le type ne change pas")
		assert.NotContains(t, body, "Changer de type")
		assert.Contains(t, body, `<option value="`+sender+`" selected>`+sender+`</option>`)
		assert.Contains(t, body, `<option value="`+template+`" selected>`+template+`</option>`)
		assert.Contains(t, body, `value="exploitation@example.org"`)

		body = s.get(t, permWrite, path(s.orders, "/tasks/error/0/edit")).Body.String()
		assert.Contains(t, body, ">lot=A\nvide=</textarea>", "the SETINFO keys one a line")

		assert.Equal(t, http.StatusNotFound, s.get(t, permWrite, path(s.orders, "/tasks/post/3/edit")).Code)
	})

	t.Run("Given a task with parameters the screen does not know", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "inconnus", IsSend: true, Path: "inconnus"}
		require.NoError(t, s.db.Insert(rule).Run())
		require.NoError(t, s.db.Insert(&model.Task{
			RuleID: rule.ID, Chain: model.ChainPre, Type: tasks.Copy, Args: map[string]string{"path": "/a", "ancien": "x"},
		}).Run())

		body := s.get(t, permWrite, path(rule, "/tasks/pre/0/edit")).Body.String()
		assert.Contains(t, body, "Paramètres conservés tels quels, que cet écran ne connaît pas : ancien.")

		rec := s.post(t, permWrite, s.at(t, rule, model.ChainPre, "pre", 0, "/edit"), url.Values{"type": {"MOVE"}, "copy.path": {"/b"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=task_updated")

		got := s.chain(t, rule, model.ChainPre)
		assert.Equal(t, []string{tasks.Copy}, typesOf(got), "the type is kept")
		assert.Equal(t, map[string]string{"path": "/b", "ancien": "x"}, map[string]string(got[0].Args))
	})

	t.Run("Given a SETINFO task", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "infos", IsSend: true, Path: "infos"}
		require.NoError(t, s.db.Insert(rule).Run())
		require.NoError(t, s.db.Insert(&model.Task{
			RuleID: rule.ID, Chain: model.ChainPost, Type: tasks.SetInfo, Args: map[string]string{"lot": "A"},
		}).Run())

		rec := s.post(t, permWrite, s.at(t, rule, model.ChainPost, "post", 0, "/edit"),
			url.Values{"setinfo.pairs": {"lot = B\r\n\r\nvide=\n"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, map[string]string{"lot": "B", "vide": ""}, map[string]string(s.chain(t, rule, model.ChainPost)[0].Args),
			"the lines replace the keys; an empty value stays (it removes the key when the task runs)")
	})

	t.Run("Given a task of a type the screen does not know", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusNotFound, s.get(t, permWrite, "/webui/rules/9999/tasks/post/0/edit").Code)
	})
}

func TestTaskDelete(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	rec := s.get(t, permWrite, path(s.orders, "/tasks/post/0/delete"))
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Supprimer la tâche n° 1 (après le transfert) ?")
	assert.Contains(t, body, "les tâches suivantes remontent d&#39;un rang")
	assert.Contains(t, body, "/archives")

	rec = s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 0, "/delete"), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=task_deleted")
	assert.Equal(t, []string{tasks.Email, tasks.Move}, typesOf(s.chain(t, s.orders, model.ChainPost)),
		"the next ones move up one rank")
}

func TestTaskOrder(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	at := func() string {
		return path(s.orders, "/tasks/post/order") + "?v=" + versionOf(s.chain(t, s.orders, model.ChainPost))
	}
	ranks := func(r ...string) url.Values { return url.Values{orderField: r} }

	rec := s.post(t, permWrite, at(), ranks("2", "0", "1"))
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=tasks_ordered")
	assert.Equal(t, []string{tasks.Move, tasks.Copy, tasks.Email}, typesOf(s.chain(t, s.orders, model.ChainPost)),
		"the chain in the order dragged")

	for _, tc := range []struct {
		name string
		form url.Values
	}{
		{"a rank missing", ranks("0", "1")},
		{"a rank twice", ranks("0", "0", "1")},
		{"a rank the chain does not have", ranks("0", "1", "3")},
		{"a rank that is not one", ranks("0", "1", "x")},
	} {
		assert.Equal(t, http.StatusBadRequest, s.post(t, permWrite, at(), tc.form).Code, tc.name)
	}

	assert.Equal(t, http.StatusBadRequest, s.post(t, permWrite, path(s.orders, "/tasks/post/order"), ranks("0", "1", "2")).Code,
		"an order without the state of the chain shown")

	rec = s.post(t, permWrite, path(s.orders, "/tasks/post/order")+"?v=0123456789abcdef", ranks("1", "0", "2"))
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=task_stale", "a chain changed since the page showed it")
	assert.Equal(t, []string{tasks.Move, tasks.Copy, tasks.Email}, typesOf(s.chain(t, s.orders, model.ChainPost)),
		"nothing moved")
}

func TestTaskMove(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.taskFixtures(t)

	rec := s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 2, "/up"), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=task_moved")
	assert.Equal(t, []string{tasks.Copy, tasks.Move, tasks.Email}, typesOf(s.chain(t, s.orders, model.ChainPost)))

	rec = s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 0, "/down"), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, []string{tasks.Move, tasks.Copy, tasks.Email}, typesOf(s.chain(t, s.orders, model.ChainPost)))

	rec = s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 0, "/up"), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.NotContains(t, rec.Header().Get("Location"), "done=", "the first one stays where it is")
	assert.Equal(t, []string{tasks.Move, tasks.Copy, tasks.Email}, typesOf(s.chain(t, s.orders, model.ChainPost)))

	assert.Equal(t, http.StatusNotFound, s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 3, "/up"), url.Values{}).Code,
		"a rank the chain shown does not have")
}

// TestExecLimit: the time limit of an EXEC task reads as a duration, on the
// folded line with the program, "no limit" when absent; the form asks it as
// a duration and stores the milliseconds the product reads (#784).
func TestExecLimit(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rule := &model.Rule{Name: "limite", IsSend: true, Path: "limite"}
	require.NoError(t, s.db.Insert(rule).Run())

	for _, task := range []*model.Task{
		{RuleID: rule.ID, Chain: model.ChainPost, Rank: 0, Type: tasks.Exec, Args: map[string]string{
			"path": "/opt/scripts/rangement.sh", "args": "--quiet", "delay": "90000",
		}},
		{RuleID: rule.ID, Chain: model.ChainPost, Rank: 1, Type: tasks.Exec, Args: map[string]string{"path": `C:\scripts\trace.bat`}},
	} {
		require.NoError(t, s.db.Insert(task).Run())
	}

	body := s.get(t, permRead, path(rule, "/tasks")).Body.String()
	assert.Contains(t, body, `<strong class="rules-taskname">EXEC</strong> <span class="rules-taskmeta">rangement.sh · 1 min 30 s</span>`,
		"the program and the limit on the folded line")
	assert.Contains(t, body, `<strong class="rules-taskname">EXEC</strong> <span class="rules-taskmeta">trace.bat · Sans limite</span>`,
		"a program without a limit, whatever the separators of its path")
	assert.Contains(t, body, `<span class="rules-value">1 min 30 s</span>`)
	assert.Contains(t, body, `<span class="rules-value">Sans limite</span>`, "the absent limit is said")
	assert.NotContains(t, body, "90000", "never the milliseconds")

	body = s.get(t, permWrite, path(rule, "/tasks/post/0/edit")).Body.String()
	assert.Contains(t, body, "Durée maximale d&#39;exécution")
	assert.Contains(t, body, `value="1m30s"`, "the form shows the limit as a duration")
	assert.Contains(t, body, "Au-delà, le programme est interrompu et le transfert passe en erreur. Vide : sans limite. Une durée : 30s, 5m, 1h30m.")

	edit := func(limit string) int {
		return s.post(t, permWrite, s.at(t, rule, model.ChainPost, "post", 0, "/edit"),
			url.Values{"exec.path": {"/opt/scripts/rangement.sh"}, "exec.args": {"--quiet"}, "exec.delay": {limit}}).Code
	}

	require.Equal(t, http.StatusSeeOther, edit("2m"))
	assert.Equal(t, "120000", s.chain(t, rule, model.ChainPost)[0].Args["delay"], "stored in milliseconds")

	require.Equal(t, http.StatusSeeOther, edit("2m"), "the duration shown, posted back")
	assert.Equal(t, "120000", s.chain(t, rule, model.ChainPost)[0].Args["delay"], "a value left as shown keeps what is stored")

	require.Equal(t, http.StatusSeeOther, edit(""))
	assert.NotContains(t, s.chain(t, rule, model.ChainPost)[0].Args, "delay", "empty: no limit, nothing stored")

	assert.Equal(t, http.StatusUnprocessableEntity, edit("bientôt"), "not a duration")
}
