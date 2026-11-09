package batch

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// The bench of the grouped actions (#792): how long the resumption, then the
// cancellation, of a set of failed transfers take, on the base the
// environment names. Skipped unless GW_BATCH_SIZES gives the sizes of the sets
// ("1000,10000"). GW_BATCH_DB names the base (sqlite by default, postgresql or
// mysql), GW_BATCH_ADDR its address, GW_BATCH_USER, GW_BATCH_PASSWORD and
// GW_BATCH_NAME the credentials and the name of an empty base of the bench,
// never one of the product.
const (
	envSizes    = "GW_BATCH_SIZES"
	envDB       = "GW_BATCH_DB"
	envAddr     = "GW_BATCH_ADDR"
	envUser     = "GW_BATCH_USER"
	envPassword = "GW_BATCH_PASSWORD"
	envName     = "GW_BATCH_NAME"
)

// volumeDB is the base of the bench: the in-memory SQLite of the tests, or
// the server the environment names.
func volumeDB(t *testing.T) *database.DB {
	t.Helper()

	config := &conf.ServerConfig{}
	config.GatewayName = "gw-bench"
	config.Paths.FilePerms = 0o600
	config.Paths.DirPerms = 0o700
	config.Database = conf.DatabaseConfig{
		Type:     os.Getenv(envDB),
		Address:  os.Getenv(envAddr),
		User:     os.Getenv(envUser),
		Password: os.Getenv(envPassword),
		Name:     os.Getenv(envName),
	}

	if config.Database.Type == "" {
		config.Database.Type = database.SQLite
		config.Database.Address = t.TempDir() + "/bench.db"
	}

	// The key of the secrets of the base: a test one, as the base of the tests.
	block, err := aes.NewCipher(make([]byte, 32)) //nolint:mnd // the size of an AES-256 key
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	db := &database.DB{Logger: logtest.GetTestLogger(t, logtest.WithLevel("WARNING")), Config: config}
	db.ChangeAEAD(gcm)
	require.NoError(t, db.Start(), "cannot start the base of the bench")

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		require.NoError(t, db.Stop(ctx))
	})

	return db
}

// TestVolume inserts n failed transfers, resumes them all in one gesture,
// then cancels them all, and logs the durations.
func TestVolume(t *testing.T) {
	sizes := os.Getenv(envSizes)
	if sizes == "" {
		t.Skipf("the bench runs when %s names the sizes of the sets", envSizes)
	}

	db := volumeDB(t)
	logger := testhelpers.GetTestLogger(t)

	partner := &model.RemoteAgent{Name: "banque", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	require.NoError(t, db.Insert(partner).Run())

	account := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "prod"}
	require.NoError(t, db.Insert(account).Run())

	client := &model.Client{Name: "cli", Protocol: "sftp"}
	require.NoError(t, db.Insert(client).Run())

	rule := &model.Rule{Name: "releves", IsSend: true, Path: "/releves"}
	require.NoError(t, db.Insert(rule).Run())

	for _, field := range strings.Split(sizes, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(field))
		require.NoError(t, err)

		ids := insertFailed(t, db, rule, client, account, n)

		start := time.Now()
		resumed, err := ResumeSet(db, ids, time.Now())
		require.NoError(t, err)
		require.Len(t, resumed.Done, n)
		require.Empty(t, resumed.Refused)
		t.Logf("%s · resume %d: %s", config(db), n, time.Since(start).Round(time.Millisecond))

		start = time.Now()
		cancelled, err := CancelSet(context.Background(), db, logger, ids)
		require.NoError(t, err)
		require.Len(t, cancelled.Done, n)
		require.Empty(t, cancelled.Refused)
		t.Logf("%s · cancel %d: %s", config(db), n, time.Since(start).Round(time.Millisecond))

		require.NoError(t, db.DeleteAll(&model.HistoryEntry{}).In("id", utils.AsAny(ids)...).Run())
	}
}

func config(db *database.DB) string {
	return db.Config.Database.Type
}

// insertFailed inserts n failed client transfers, by batches, and gives their
// identifiers.
func insertFailed(t *testing.T, db *database.DB, rule *model.Rule, client *model.Client,
	account *model.RemoteAccount, n int,
) []int64 {
	t.Helper()

	const batch = 500

	ids := make([]int64, 0, n)
	start := time.Now()

	for from := 0; from < n; from += batch {
		transfers := make([]*model.Transfer, 0, batch)

		for i := from; i < min(from+batch, n); i++ {
			transfers = append(transfers, &model.Transfer{
				RuleID: rule.ID, ClientID: client.NullableID(), RemoteAccountID: account.NullableID(),
				SrcFilename: "bench-" + strconv.Itoa(i) + ".pdf", DestFilename: "bench-" + strconv.Itoa(i) + ".pdf",
				Status: types.StatusError, ErrCode: types.TeConnection, ErrDetails: "connection refused",
				Step: types.StepSetup, Start: time.Now(), Filesize: 1024,
				TransferInfo: map[string]any{"lot": "bench", model.FollowID: int64(i)},
			})
		}

		require.NoError(t, database.InsertBatch(db, transfers...))

		for _, trans := range transfers {
			ids = append(ids, trans.ID)
		}
	}

	t.Logf("%s · insert %d: %s", config(db), n, time.Since(start).Round(time.Millisecond))

	return ids
}
