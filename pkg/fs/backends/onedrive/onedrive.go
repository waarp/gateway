package onedrive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/rclone/rclone/backend/onedrive"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"
	"golang.org/x/oauth2"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/internal"
)

var ErrMissingDriveID = errors.New(`missing onedrive "drive_id" parameter`)

func NewFS(name, clientID, clientSecret string, userOpts map[string]string) (fs.FS, error) {
	token, err := makeTokenFunc(clientID, clientSecret, configmap.Simple(userOpts))
	if err != nil {
		return nil, err
	}

	odVFS, err := newVFS(name, clientID, token, clientSecret, userOpts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: odVFS}, nil
}

func newVFS(name, clientID, clientSecret, token string, userOpts map[string]string) (*vfs.VFS, error) {
	config, err := parseOpts(clientID, clientSecret, token, userOpts)
	if err != nil {
		return nil, err
	}

	odfs, err := onedrive.NewFs(context.Background(), name, "", config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate onedrive filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()

	return vfs.New(context.Background(), odfs, vfsOpts), nil
}

type tokenRefresher struct {
	m            configmap.Mapper
	clientID     string
	clientSecret string
	mu           sync.Mutex
}

//nolint:gochecknoglobals //var is needed here for testing
var makeTokenFunc = makeToken

func (t *tokenRefresher) Get(key string) (string, bool) {
	if key != "token" {
		return t.m.Get(key)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if val, ok := t.m.Get(key); ok && val != "" {
		var token oauth2.Token
		if err := json.Unmarshal([]byte(val), &token); err == nil {
			if token.Valid() || token.RefreshToken != "" {
				return val, true
			}
		}
	}

	newToken, err := makeTokenFunc(t.clientID, t.clientSecret, t.m)
	if err != nil {
		return "", false
	}

	t.m.Set(key, newToken)

	return newToken, true
}

func (t *tokenRefresher) Set(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.m.Set(key, value)
}

func parseOpts(clientID, clientSecret, token string, userOpts map[string]string,
) (configmap.Mapper, error) {
	// Credential options
	credOpts := configmap.Simple{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"token":         token,
	}

	// Make config map
	config := internal.MakeOptions("onedrive", userOpts, credOpts)

	// Check mandatory values
	if drive, _ := config.Get("drive_id"); drive == "" {
		return nil, ErrMissingDriveID
	}

	return &tokenRefresher{
		m:            config,
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

func makeToken(clientID, clientSecret string, opts configmap.Getter) (string, error) {
	const tenantKey = "tenant"

	cred, err := confidential.NewCredFromSecret(clientSecret)
	if err != nil {
		return "", fmt.Errorf("failed to create MSAL credential: %w", err)
	}

	tenant, _ := opts.Get(tenantKey)
	client, err := confidential.New("https://login.microsoftonline.com/"+tenant, clientID, cred)
	if err != nil {
		return "", fmt.Errorf("failed to create MSAL client: %w", err)
	}

	scope := []string{"https://graph.microsoft.com/.default"}
	result, err := client.AcquireTokenByCredential(context.Background(), scope)
	if err != nil {
		return "", fmt.Errorf("failed to acquire access token: %w", err)
	}

	token := oauth2.Token{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		Expiry:      result.ExpiresOn,
	}
	raw, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	return string(raw), nil
}
