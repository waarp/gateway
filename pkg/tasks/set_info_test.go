package tasks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestSetInfoOld(t *testing.T) {
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

	t.Run("Add number", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"key": "newKey", "value": "123"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{"existingKey": "existingValue", "newKey": json.Number("123")},
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
		params := map[string]string{"key": "existingKey"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{},
			transCtx.Transfer.TransferInfo,
		)
	})
}

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
		params := map[string]string{"newKey1": "newValue1", "newKey2": "newValue2"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{"existingKey": "existingValue", "newKey1": "newValue1", "newKey2": "newValue2"},
			transCtx.Transfer.TransferInfo,
		)
	})

	t.Run("Add number", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"newKey": "123"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{"existingKey": "existingValue", "newKey": json.Number("123")},
			transCtx.Transfer.TransferInfo,
		)
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"existingKey": "newValue"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{"existingKey": "newValue"},
			transCtx.Transfer.TransferInfo,
		)
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"existingKey": "null"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil))
		assert.Equal(t,
			map[string]any{},
			transCtx.Transfer.TransferInfo,
		)
	})
}
