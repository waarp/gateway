//go:build manual_test

package gui

import (
	"fmt"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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
	// Here, for example, we insert a new user in the database.
	user := &model.User{Username: "toto", PasswordHash: "<PASSWORD>"}
	require.NoError(t, internal.InsertUser(db, user))

	// Instantiate the server with the handler you wish to test.
	server := httptest.NewServer(homepage(db, logger))
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
