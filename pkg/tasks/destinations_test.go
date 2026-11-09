package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The table of the task destinations the disk measure reads names
// tasks the gateway registers, and only arguments those tasks read.
func TestAnalyticsDestinations(t *testing.T) {
	t.Parallel()

	args := map[string]string{
		Copy: "path", CopyRename: "path", Move: "path", MoveRename: "path", Rename: "path",
		Archive: "outputPath", Extract: "outputDir",
		Encrypt: "outputFile", Decrypt: "outputFile", Sign: "outputFile",
		EncryptAndSign: "outputFile", DecryptAndVerify: "outputFile",
		Transfer: "file", Preregister: "file",
	}

	require.Len(t, analytics.TaskDestinations, len(args), "every task that writes a local destination, and no other")

	for typ, dest := range analytics.TaskDestinations {
		_, registered := model.ValidTasks[typ]
		assert.Truef(t, registered, "%q is not a task of the gateway", typ)
		assert.Equalf(t, args[typ], dest.Arg, "%q: the argument of its destination", typ)
	}

	// A directory argument names the directory itself, a file argument the
	// file: the measure takes the directory of the latter.
	for _, typ := range []string{Copy, Move, Extract} {
		assert.Truef(t, analytics.TaskDestinations[typ].Dir, "%q writes into a directory", typ)
	}

	for _, typ := range []string{CopyRename, MoveRename, Rename, Archive, Encrypt, Transfer, Preregister} {
		assert.Falsef(t, analytics.TaskDestinations[typ].Dir, "%q names a file", typ)
	}
}
