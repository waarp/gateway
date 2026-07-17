package tasks

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetInfo(t *testing.T) {
	t.Parallel()
	logger := logtest.GetTestLogger(t)

	mkCtx := func() *model.TransferContext {
		return &model.TransferContext{
			Transfer: &model.Transfer{
				TransferInfo: map[string]any{
					"existingKey": "existingValue",
				},
			},
		}
	}

	t.Run("Add", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"key": "newKey", "value": "newValue"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{"existingKey": "existingValue", "newKey": "newValue"},
			transCtx.Transfer.TransferInfo,
		)
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"key": "existingKey", "value": "newValue"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{"existingKey": "newValue"},
			transCtx.Transfer.TransferInfo,
		)
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"key": "existingKey", "value": ""}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{},
			transCtx.Transfer.TransferInfo,
		)
	})
}
