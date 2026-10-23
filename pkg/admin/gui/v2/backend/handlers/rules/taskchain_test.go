package rules

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// assertStale asserts that a change was refused: the tab, saying nothing was
// saved.
func assertStale(t *testing.T, rec interface{ Result() *http.Response }, rule *model.Rule) {
	t.Helper()

	res := rec.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	assert.Contains(t, res.Header.Get("Location"), path(rule, "/tasks"))
	assert.Contains(t, res.Header.Get("Location"), "done=task_stale")
}

// A change sent from a page that no longer shows its chain is refused: it
// never applies to another task than the one the page showed. The tab comes
// back with its current state and says that nothing was saved.
func TestTaskStale(t *testing.T) {
	t.Parallel()

	t.Run("Given a deletion confirmed after the task was deleted", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.taskFixtures(t) // after the transfer: COPY, EMAIL, MOVE

		page := s.at(t, s.orders, model.ChainPost, "post", 0, "/delete")
		require.Equal(t, http.StatusOK, s.get(t, permWrite, page).Code)
		require.Equal(t, http.StatusSeeOther,
			s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 0, "/delete"), url.Values{}).Code)

		assertStale(t, s.post(t, permWrite, page, url.Values{}), s.orders)
		assert.Equal(t, []string{tasks.Email, tasks.Move}, typesOf(s.chain(t, s.orders, model.ChainPost)),
			"the task now at that rank stays")
	})

	t.Run("Given an edition saved after the chain changed", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.taskFixtures(t)

		page := s.at(t, s.orders, model.ChainPost, "post", 2, "/edit") // MOVE
		require.Equal(t, http.StatusSeeOther,
			s.post(t, permWrite, s.at(t, s.orders, model.ChainPost, "post", 2, "/up"), url.Values{}).Code)

		assertStale(t, s.get(t, permWrite, page), s.orders)
		assertStale(t, s.post(t, permWrite, page, url.Values{"copy.path": {"/nouveau"}}), s.orders)

		got := s.chain(t, s.orders, model.ChainPost)
		assert.Equal(t, []string{tasks.Copy, tasks.Move, tasks.Email}, typesOf(got))
		assert.Equal(t, map[string]string{"path": "/sortie"}, map[string]string(got[1].Args), "the task is not changed")
		assert.Equal(t, map[string]string{"path": "/archives"}, map[string]string(got[0].Args), "nor another one")
	})

	t.Run("Given a move sent after a task was added", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.taskFixtures(t)

		page := s.at(t, s.orders, model.ChainPost, "post", 0, "/down")
		require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, path(s.orders, "/tasks/post/new"),
			url.Values{"type": {tasks.Delete}}).Code, "an addition carries no version: it goes at the end")

		assertStale(t, s.post(t, permWrite, page, url.Values{}), s.orders)
		assert.Equal(t, []string{tasks.Copy, tasks.Email, tasks.Move, tasks.Delete},
			typesOf(s.chain(t, s.orders, model.ChainPost)))
	})

	t.Run("Given a change without the state of its chain", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.taskFixtures(t)

		for _, suffix := range []string{"/delete", "/edit", "/up", "/down"} {
			assertStale(t, s.post(t, permWrite, path(s.orders, "/tasks/post/1"+suffix), url.Values{}), s.orders)
		}

		assert.Equal(t, []string{tasks.Copy, tasks.Email, tasks.Move}, typesOf(s.chain(t, s.orders, model.ChainPost)))
	})

	t.Run("Given a chain changed between the read of a request and its write", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.taskFixtures(t)

		h := New(s.db, testhelpers.GetTestLogger(t))
		shown := versionOf(s.chain(t, s.orders, model.ChainPost))
		require.NoError(t, h.modifyChain(s.orders.ID, model.ChainPost, "", func(current []*model.Task) []*model.Task {
			return current[1:]
		}), "a change without a version applies to the chain as the transaction reads it")

		err := h.modifyChain(s.orders.ID, model.ChainPost, shown, func([]*model.Task) []*model.Task { return nil })
		require.ErrorIs(t, err, errChainChanged, "the chain is read again in the transaction")
		assert.Equal(t, []string{tasks.Email, tasks.Move}, typesOf(s.chain(t, s.orders, model.ChainPost)), "nothing written")
	})

	t.Run("Given the tab after a refused change", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, path(s.orders, "/tasks")+"?done=task_stale").Body.String()
		assert.Contains(t, body, "w-toast--ko", "an error, which stays")
		assert.Contains(t, body, "Rien n&#39;a été enregistré")
		assert.Contains(t, body, "La chaîne a changé depuis l&#39;affichage de la page : voici son état actuel.")
	})
}
