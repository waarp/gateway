package clouds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The options of an instance that are never shown in clear: the token and
// the former credentials, and what the storage library masks among the
// options the form does not ask. What the form asks and the sheet shows
// stays.
func TestSecretOptions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"secret_access_key", "session_token", "sse_customer_key"},
		SecretOptions("s3", map[string]string{
			"region": "eu-west-3", "session_token": "t", "sse_customer_key": "k", "secret_access_key": "former",
		}))

	assert.Equal(t, []string{"token"}, SecretOptions("onedrive", map[string]string{
		"token": "oauth", "drive_id": "b!id", "tenant": "contoso",
	}), "the identifiers the sheet shows are not secrets")

	assert.Equal(t, []string{"token"}, SecretOptions("gcs", map[string]string{"token": "oauth", "project_number": "42"}))

	assert.Empty(t, SecretOptions("s3", map[string]string{"region": "eu-west-3"}))
	assert.Empty(t, SecretOptions("s3", nil))

	// A storage type this screen does not know: nothing says what is a
	// secret, every option is one.
	assert.Equal(t, []string{"a", "b"}, SecretOptions("unknown", map[string]string{"b": "2", "a": "1"}))
}
