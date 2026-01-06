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

func NewFS(name, clientID, clientSecret string, opts map[string]string) (fs.FS, error) {
	token, err := makeTokenFunc(clientID, clientSecret, configmap.Simple(opts))
	if err != nil {
		return nil, err
	}

	opts["token"] = token

	odVFS, err := newVFS(name, clientID, clientSecret, opts)
	if err != nil {
		return nil, err
	}

	return &fs.VFS{VFS: odVFS}, nil
}

func newVFS(name, clientID, clientSecret string, optsMap map[string]string) (*vfs.VFS, error) {
	odOpts, err := parseOpts(clientID, clientSecret, optsMap)
	if err != nil {
		return nil, err
	}

	odfs, err := onedrive.NewFs(context.Background(), name, "", odOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate onedrive filesystem: %w", err)
	}

	vfsOpts := internal.VFSOpts()

	return vfs.New(odfs, vfsOpts), nil
}

type tokenRefresher struct {
	m            configmap.Simple
	clientID     string
	clientSecret string
	mu           sync.Mutex
}

//nolint:gochecknoglobals //var is needed here for testing
var makeTokenFunc = makeToken

func (t *tokenRefresher) Get(key string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if key == "token" {
		if val, ok := t.m.Get(key); ok && val != "" {
			var token oauth2.Token
			if err := json.Unmarshal([]byte(val), &token); err == nil {
				if token.Valid() || token.RefreshToken != "" {
					return val, true
				}
			}
		}

		newToken, err := makeTokenFunc(t.clientID, t.clientSecret, t.m)
		if err == nil {
			t.m.Set(key, newToken)

			return newToken, true
		}
	}

	return t.m.Get(key)
}

func (t *tokenRefresher) Set(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.m.Set(key, value)
}

func parseOpts(clientID, clientSecret string, opts map[string]string) (configmap.Mapper, error) {
	const (
		regionKey    = "region"
		driveIDKey   = "drive_id"
		driveTypeKey = "drive_type"
	)

	internal.SetDefaultValue(opts, "chunk_size", "10Mi")
	internal.SetDefaultValue(opts, driveTypeKey, "business")
	internal.SetDefaultValue(opts, regionKey, "global")

	if opts[driveIDKey] == "" {
		return nil, ErrMissingDriveID
	}

	if clientID != "" {
		opts["client_id"] = clientID
	}

	if clientSecret != "" {
		opts["client_secret"] = clientSecret
	}

	return &tokenRefresher{
		m:            opts,
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
