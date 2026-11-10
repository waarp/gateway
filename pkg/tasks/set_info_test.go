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

	t.Run("Mixed with other keys", func(t *testing.T) {
		t.Parallel()
		task, transCtx := &setInfoTask{}, mkCtx()
		params := map[string]string{"key": "newKey", "value": "newValue", "otherKey": "dropped"}

		require.NoError(t, task.Run(t.Context(), params, nil, logger, transCtx, nil),
			"a stored task of a mixed form keeps running as before: the one key of the deprecated form")
		assert.Equal(t,
			map[string]any{"existingKey": "existingValue", "newKey": "newValue"},
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

func TestSetInfoPairs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]string{"k": "v"}, SetInfoPairs(map[string]string{"key": "k", "value": "v"}),
		"the deprecated form sets the one key it names")
	assert.Equal(t, map[string]string{"k": ""}, SetInfoPairs(map[string]string{"key": "k"}),
		"without a value, the deprecated form removes its key")
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, SetInfoPairs(map[string]string{"a": "1", "b": "2"}),
		"the current form is read as it is")
	assert.Equal(t, map[string]string{"k": "v"}, SetInfoPairs(map[string]string{"key": "k", "value": "v", "a": "1"}),
		"a mixed form is read as the deprecated one: its other keys are dropped")

	assert.True(t, SetInfoReserved("key"))
	assert.True(t, SetInfoReserved("value"))
	assert.False(t, SetInfoReserved("__followID__"))
}

func TestSetInfoValidate(t *testing.T) {
	t.Parallel()

	for name, args := range map[string]map[string]string{
		"the current form":              {"a": "1", "b": "2"},
		"the deprecated form":           {"key": "k", "value": "v"},
		"the deprecated form, no value": {"key": "k"},
	} {
		assert.NoError(t, (&setInfoTask{}).Validate(args), name)
	}

	for name, args := range map[string]map[string]string{
		"a mixed form":                    {"key": "k", "value": "v", "a": "1"},
		"a key named value":               {"value": "v"},
		"a key named value, among others": {"a": "1", "value": "v"},
	} {
		assert.ErrorIs(t, (&setInfoTask{}).Validate(args), ErrSetInfoReservedArgs, name)
	}
}
