package internal

import (
	"context"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"github.com/stretchr/testify/assert"
)

type dummyPipeline struct{ paused, cancelled bool }

func (d *dummyPipeline) Pause(context.Context) error {
	d.paused = true

	return nil
}

func (d *dummyPipeline) Cancel(context.Context) error {
	d.cancelled = true

	return nil
}

func TestR66Error(t *testing.T) {
	for code := types.TeOk; code.IsValid(); code++ {
		t.Run(code.String(), func(t *testing.T) {
			origErr := pipeline.NewError(code, "error message")
			r66err := ToR66Error(origErr)
			assert.Equal(t, string(code.R66Code()), string(r66err.Code))

			pip := &dummyPipeline{}
			destErr := FromR66Error(r66err, pip)

			switch code {
			case types.TeOk:
				assert.Nil(t, destErr)
			case types.TeStopped:
				assert.True(t, pip.paused)
			case types.TeCanceled:
				assert.True(t, pip.cancelled)
			case types.TeExpired:
				assert.Equal(t, types.TeUnknown, destErr.Code())
			default:
				assert.Equal(t, origErr.Code().String(), destErr.Code().String())
				assert.Equal(t, "Error on remote partner: "+origErr.Details(), destErr.Details())
			}
		})
	}
}
