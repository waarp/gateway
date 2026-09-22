package pesit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestWaitAckClient(t *testing.T) {
	db := dbtest.TestDatabase(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, map[string]any{
		"expectsAck": true, "ackTimeout": "10s",
	})

	waiting := make(chan struct{})
	pip := ctx.PushPipeline(t)
	pip.Pip.Trace.OnClose = func() error {
		close(waiting)

		return nil
	}

	defer pip.Cancel(t.Context())
	go pip.Run()

	<-waiting

	var check model.NormalizedTransferView
	require.NoError(t, db.Get(&check, "id=?", ctx.TransferPush.ID).Eager().Run())

	assert.True(t, check.IsTransfer)
	assert.Subset(t, check.TransferInfo, map[string]any{
		ackExpectedKey: true, ackTimeout: "10s",
	})
}
