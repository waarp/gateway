package login

import (
	"bytes"
	"context"
	"testing"

	"code.waarp.fr/lib/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bracket is a translator that shows the key, as the tests of the kit do.
func bracket(key string, _ ...any) string { return "[" + key + "]" }

func TestSentences(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "[login.title]", Title(bracket))
	assert.Equal(t, "[login.refused]", Refusal(bracket, true), "wrong credentials or a locked account")
	assert.Equal(t, "[login.unavailable]", Refusal(bracket, false), "the check could not be done")
	assert.Equal(t, "[login.expired]", Info(bracket, ReasonExpired))
	assert.Equal(t, "[login.signed_out]", Info(bracket, ReasonSignedOut))
	assert.Empty(t, Info(bracket, ""), "no reason")
	assert.Empty(t, Info(bracket, "whatever"), "an unknown reason")
}

// The page keeps the username and never a password; it shows the refusal as
// an alert and the information about the session as a status, each only when
// given.
func TestPage(t *testing.T) {
	t.Parallel()

	render := func(d Data) string {
		var out bytes.Buffer
		require.NoError(t, gui.Render(context.Background(), &out, bracket,
			gui.Options{BasePath: "/webui", StaticPath: "/static/gui"}, Page(d)))

		return out.String()
	}

	html := render(Data{})
	for _, want := range []string{
		`src="/webui/static/gui/waarp-banner.png"`, `<h1 class="w-plain__title">[login.title]</h1>`,
		`<form class="w-form" method="post">`, `for="username">[login.username]</label>`,
		`id="username" name="username" value="" required autofocus autocomplete="username" autocapitalize="none" spellcheck="false">`,
		`for="password">[login.password]</label>`,
		`<div class="w-password"><input class="w-input" type="password" id="password" name="password" autocomplete="current-password" required>`,
		`<button class="w-btn w-btn--primary w-plain__submit" type="submit">[login.title]</button>`,
	} {
		assert.Contains(t, html, want)
	}

	assert.NotContains(t, html, `role="alert"`)
	assert.NotContains(t, html, `role="status"`)

	html = render(Data{Username: "exploitant", Error: "Refusé."})
	assert.Contains(t, html, `name="username" value="exploitant"`)
	assert.Contains(t, html, `<div class="w-banner w-banner--ko" role="alert">`)
	assert.Contains(t, html, `<strong>Refusé.</strong>`)

	html = render(Data{Info: "Session terminée."})
	assert.Contains(t, html, `<div class="w-banner w-banner--info" role="status">`)
	assert.Contains(t, html, `<strong>Session terminée.</strong>`)
}
