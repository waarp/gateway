package rules

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func itoa64(id int64) string { return strconv.FormatInt(id, 10) }

func TestAccessTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given a rule open to all", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, path(s.receive, "/access"))
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, `href="/webui/rules/`+itoa64(s.receive.ID)+`/access" aria-current="page">Accès`)
		assert.Contains(t, body, "<strong>Ouverte à tous</strong>")
		assert.Contains(t, body, "Ajouter un accès")
		assert.NotContains(t, body, "Rouvrir à tous", "nothing to open")
		assert.NotContains(t, body, "Quand un partenaire se connecte", "no block without access")
	})

	t.Run("Given a rule kept for a partner", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, path(s.send, "/access")).Body.String()
		assert.Contains(t, body, "<strong>Réservée</strong>")
		assert.Contains(t, body, "Quand la passerelle se connecte")
		assert.Contains(t, body, ">banque-demo<")
		assert.Contains(t, body, "Aucun serveur ni compte local.")
		assert.Contains(t, body, "La règle n&#39;est réservée que d&#39;un côté.", "the other side is closed too")
		assert.Contains(t, body, "Rouvrir à tous")
		assert.Contains(t, body, "Retirer l&#39;accès")

		body = s.get(t, permRead, path(s.send, "/access")).Body.String()
		assert.NotContains(t, body, "Ajouter un accès", "no addition without the write right")
		assert.NotContains(t, body, "Retirer l&#39;accès")
		assert.NotContains(t, body, "Rouvrir à tous")
	})
}

func TestAccessAdd(t *testing.T) {
	t.Parallel()

	t.Run("Given the form of a rule open to all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.receive, "/access/new")).Body.String()
		assert.Contains(t, body, "La règle est ouverte à tous.", "the first access closes it")
		assert.Contains(t, body, `<option value="`+itoa64(s.server.ID)+`">srv-sftp</option>`)
		assert.Contains(t, body, `<optgroup label="srv-sftp"><option value="`+itoa64(s.local.ID)+`">filiale-sud</option>`)
	})

	t.Run("Given the form of a rule kept for a partner", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.send, "/access/new")).Body.String()
		assert.NotContains(t, body, ">banque-demo</option>", "a target already granted is left out")
		assert.Contains(t, body, "Tous ont déjà accès à la règle.")
	})

	t.Run("Given a local account added", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.receive, "/access/new"), url.Values{
			fieldKind: {kindLocalAccount}, kindLocalAccount: {itoa64(s.local.ID)},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), path(s.receive, "/access"))
		assert.Equal(t, uint64(1), s.count(t, &model.RuleAccess{}, "rule_id=? AND local_account_id=?", s.receive.ID, s.local.ID))
	})

	for _, tc := range []struct {
		name, message string
		form          func(s *suite) url.Values
	}{
		{"no kind", "Choisissez un type.", func(*suite) url.Values { return url.Values{} }},
		{"no target", "Choisissez la cible.", func(*suite) url.Values { return url.Values{fieldKind: {kindServer}} }},
		{
			"a target already granted", "Cette cible a déjà accès à la règle.",
			func(s *suite) url.Values {
				return url.Values{fieldKind: {kindPartner}, kindPartner: {itoa64(s.partner.ID)}}
			},
		},
	} {
		t.Run("Given "+tc.name, func(t *testing.T) {
			t.Parallel()

			s := newSuite(t)
			s.fixtures(t)

			rec := s.post(t, permWrite, path(s.send, "/access/new"), tc.form(s))
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.message)
			assert.Equal(t, uint64(1), s.count(t, &model.RuleAccess{}, "rule_id=?", s.send.ID), "nothing added")
		})
	}
}

func TestAccessRemove(t *testing.T) {
	t.Parallel()

	t.Run("Given the last access of a rule", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		target := path(s.send, "/access/"+kindPartner+"/"+itoa64(s.partner.ID)+"/delete")
		body := s.get(t, permWrite, target).Body.String()
		assert.Contains(t, body, "C&#39;est le dernier accès de la règle.", "the rule opens to all")
		assert.Contains(t, body, ">Supprimer</button>")

		rec := s.post(t, permWrite, target, nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=access_removed")
		assert.Zero(t, s.count(t, &model.RuleAccess{}, "rule_id=?", s.send.ID))
	})

	t.Run("Given an access the rule does not give", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, path(s.send, "/access/"+kindServer+"/"+itoa64(s.server.ID)+"/delete"))
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Accès introuvable")

		rec = s.post(t, permWrite, path(s.send, "/access/"+kindServer+"/"+itoa64(s.server.ID)+"/delete"), nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, uint64(1), s.count(t, &model.RuleAccess{}, "rule_id=?", s.send.ID), "nothing removed")
	})
}

func TestAccessOpen(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	require.NoError(t, s.db.Insert(&model.RuleAccess{RuleID: s.send.ID, LocalAgentID: s.server.NullableID()}).Run())

	body := s.get(t, permWrite, path(s.send, "/access/open")).Body.String()
	assert.Contains(t, body, "Ses accès (2) seront retirés")

	rec := s.post(t, permWrite, path(s.send, "/access/open"), nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=access_opened")
	assert.Zero(t, s.count(t, &model.RuleAccess{}, "rule_id=?", s.send.ID), "open to all")
}
