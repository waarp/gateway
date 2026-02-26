//go:build manual_test

package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	tasknames "code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// This is an example of how to make a function for quickly a specific HTTP handler.
// This function can be called by running “go test -v -tags=manual_test -run=TestHandler“.
// The server can then be stopped when needed by sending an SIGTERM signal
// (usually done by pressing Ctrl+C in a terminal).
func TestHandler(t *testing.T) {
	// Instantiate a test logger
	logger := testhelpers.GetTestLogger(t)
	// Instantiate a test database.
	db := dbtest.TestDatabase(t)

	// If stuff needs to be added to the database, do it here.
	prepareDB(t, db)

	// Instantiate the server with the handler you wish to test.
	server := httptest.NewServer(testHandler(db, logger))
	fmt.Printf("Server started at %q\n", server.URL)
	t.Cleanup(server.Close)

	// Wait for the interruption signal to shut down the server.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	for {
		switch <-c {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			return
		}
	}
}

func prepareDB(tb testing.TB, db *database.DB) {
	tb.Helper()

	rule := &model.Rule{Name: "test_rule"}
	require.NoError(tb, db.Insert(rule).Run())

	pre1 := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   tasknames.CopyRename,
		Args:   map[string]string{"path": "/tmp/test.txt"},
	}
	require.NoError(tb, db.Insert(pre1).Run())

	pre2 := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPre,
		Rank:   1,
		Type:   tasknames.Archive,
		Args:   map[string]string{"files": "#TRUEFULLPATH#", "outputPath": "/tmp/test.zip"},
	}
	require.NoError(tb, db.Insert(pre2).Run())

	post1 := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   tasknames.Delete,
	}
	require.NoError(tb, db.Insert(post1).Run())

	err1 := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   tasknames.Exec,
		Args: map[string]string{
			"path":  "echo",
			"args":  "'hello world'",
			"delay": utils.FormatInt(90 * 60 * 1000), // 1h30m -> duration is in ms
		},
	}
	require.NoError(tb, db.Insert(err1).Run())

	email := &model.EmailTemplate{
		Name:     "example_template",
		Subject:  "Example subject",
		MIMEType: "text/plain",
		Body:     "Example email message",
	}
	require.NoError(tb, db.Insert(email).Run())

	smtpCred := &model.SMTPCredential{
		EmailAddress:  "foobar@example.com",
		ServerAddress: types.Addr("example.com", 587),
		Login:         "foobar",
		Password:      "sesame",
	}
	require.NoError(tb, db.Insert(smtpCred).Run())

	key1 := &model.CryptoKey{
		Name: "test_pgp_private_key",
		Type: model.CryptoKeyTypePGPPrivate,
		Key:  testhelpers.TestPGPPrivateKey,
	}
	require.NoError(tb, db.Insert(key1).Run())

	key2 := &model.CryptoKey{
		Name: "test_pgp_public_key",
		Type: model.CryptoKeyTypePGPPublic,
		Key:  testhelpers.TestPGPPublicKey,
	}
	require.NoError(tb, db.Insert(key2).Run())

	partner1 := &model.RemoteAgent{
		Name:     "partner1",
		Protocol: "sftp",
		Address:  types.Addr("1.1.1.1", 1111),
	}
	require.NoError(tb, db.Insert(partner1).Run())

	partner2 := &model.RemoteAgent{
		Name:     "partner2",
		Protocol: "sftp",
		Address:  types.Addr("2.2.2.2", 2222),
	}
	require.NoError(tb, db.Insert(partner2).Run())

	client1 := &model.Client{Name: "r66_client", Protocol: "r66"}
	require.NoError(tb, db.Insert(client1).Run())

	client2 := &model.Client{Name: "sftp_client", Protocol: "sftp"}
	require.NoError(tb, db.Insert(client2).Run())

	remAcc1 := &model.RemoteAccount{
		RemoteAgentID: partner1.ID,
		Login:         "remAcc1",
	}
	require.NoError(tb, db.Insert(remAcc1).Run())

	remAcc2 := &model.RemoteAccount{
		RemoteAgentID: partner1.ID,
		Login:         "remAcc2",
	}
	require.NoError(tb, db.Insert(remAcc2).Run())

	send := &model.Rule{Name: "sending", IsSend: true}
	require.NoError(tb, db.Insert(send).Run())

	recv := &model.Rule{Name: "receiving", IsSend: false}
	require.NoError(tb, db.Insert(recv).Run())

	server1 := &model.LocalAgent{Name: "server_r66", Protocol: "r66", Address: types.Addr("1.1.1.1", 1111)}
	require.NoError(tb, db.Insert(server1).Run())

	server2 := &model.LocalAgent{Name: "server_sftp", Protocol: "sftp", Address: types.Addr("2.2.2.2", 2222)}
	require.NoError(tb, db.Insert(server2).Run())

	locAcc1 := &model.LocalAccount{LocalAgentID: server2.ID, Login: "locAcc1"}
	require.NoError(tb, db.Insert(locAcc1).Run())

	locAcc2 := &model.LocalAccount{LocalAgentID: server2.ID, Login: "locAcc2"}
	require.NoError(tb, db.Insert(locAcc2).Run())
}

func testHandler(db *database.DB, logger *log.Logger) http.HandlerFunc {
	webfs.WebFS = os.DirFS("../frontend")
	dummyUser := &model.User{Username: "toto", Permissions: model.PermAll}

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, constants.StaticPrefix) {
			http.StripPrefix(constants.StaticPrefix,
				http.FileServerFS(webfs.WebFS),
			).ServeHTTP(w, r)

			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, constants.ContextUserKey, dummyUser)
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")

		router := mux.NewRouter()
		MakeListingRouter(router, db, logger)
		// Change this to the handler you wish to test.
		MakeTasksRouter(router, db, logger)

		router.ServeHTTP(w, r.WithContext(ctx))
	}
}
