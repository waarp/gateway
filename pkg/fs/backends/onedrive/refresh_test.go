package onedrive

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestTokenRefresher_Get(t *testing.T) {
	clientID := "onedrive-client-id"
	clientSecret := "onedrive-client-secret"
	tenantID := "onedrive-tenant-id"

	t.Run("Refreshes when token is missing", func(t *testing.T) {
		opts := map[string]string{"tenant": tenantID}
		refresher := &tokenRefresher{
			m:            opts,
			clientID:     clientID,
			clientSecret: clientSecret,
		}

		expectedToken := &oauth2.Token{
			AccessToken: "new-token",
			Expiry:      time.Now().Add(time.Hour),
		}
		expectedJSON, _ := json.Marshal(expectedToken)

		// Mock makeTokenFunc
		oldMakeToken := makeTokenFunc
		defer func() { makeTokenFunc = oldMakeToken }()

		callCount := 0
		makeTokenFunc = func(cid, cs string, o configmap.Getter) (string, error) {
			callCount++
			assert.Equal(t, clientID, cid)
			assert.Equal(t, clientSecret, cs)
			return string(expectedJSON), nil
		}

		val, ok := refresher.Get("token")
		assert.True(t, ok)
		assert.Equal(t, string(expectedJSON), val)
		assert.Equal(t, 1, callCount)

		// Check that it's stored in the map
		assert.Equal(t, string(expectedJSON), opts["token"])
	})

	t.Run("Returns existing token if still valid", func(t *testing.T) {
		validToken := &oauth2.Token{
			AccessToken: "valid-token",
			Expiry:      time.Now().Add(time.Hour),
		}
		validJSON, _ := json.Marshal(validToken)

		opts := map[string]string{
			"tenant": "test-tenant",
			"token":  string(validJSON),
		}
		refresher := &tokenRefresher{
			m:            opts,
			clientID:     clientID,
			clientSecret: clientSecret,
		}

		// Mock makeTokenFunc to fail if called
		oldMakeToken := makeTokenFunc
		defer func() { makeTokenFunc = oldMakeToken }()
		makeTokenFunc = func(cid, cs string, o configmap.Getter) (string, error) {
			t.Fatal("makeTokenFunc should not have been called")
			return "", nil
		}

		val, ok := refresher.Get("token")
		assert.True(t, ok)
		assert.Equal(t, string(validJSON), val)
	})

	t.Run("Refreshes when token is expired", func(t *testing.T) {
		expiredToken := &oauth2.Token{
			AccessToken: "expired-token",
			Expiry:      time.Now().Add(-time.Hour),
		}
		expiredJSON, _ := json.Marshal(expiredToken)

		opts := map[string]string{
			"tenant": "test-tenant",
			"token":  string(expiredJSON),
		}
		refresher := &tokenRefresher{
			m:            opts,
			clientID:     clientID,
			clientSecret: clientSecret,
		}

		newToken := &oauth2.Token{
			AccessToken: "new-token",
			Expiry:      time.Now().Add(time.Hour),
		}
		newJSON, _ := json.Marshal(newToken)

		// Mock makeTokenFunc
		oldMakeToken := makeTokenFunc
		defer func() { makeTokenFunc = oldMakeToken }()

		callCount := 0
		makeTokenFunc = func(cid, cs string, o configmap.Getter) (string, error) {
			callCount++
			return string(newJSON), nil
		}

		val, ok := refresher.Get("token")
		assert.True(t, ok)
		assert.Equal(t, string(newJSON), val)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, string(newJSON), opts["token"])
	})
}
