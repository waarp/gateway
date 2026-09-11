package pesit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoveryPointUsesTheNegotiatedInterval(t *testing.T) {
	t.Parallel()

	// Measured on a real peer: 10 248 192 bytes received when the gateway
	// died, interval negotiated at 36 KiB while 1 MiB was configured. The
	// peer counts its points on the negotiated value: point 278, not 9.
	const (
		progress   = 10_248_192
		negotiated = 36_864
		configured = 1_048_576
	)

	assert.Equal(t, uint32(278), recoveryPoint(progress, negotiated))
	assert.Equal(t, int64(278*negotiated), recoveryOffset(278, negotiated))
	assert.NotEqual(t, uint32(278), recoveryPoint(progress, configured),
		"the configured interval gives another point, which the peer cannot interpret")

	// The point never lies past what was received.
	assert.LessOrEqual(t, recoveryOffset(recoveryPoint(progress, negotiated), negotiated), int64(progress))

	// Without checkpoints, a transfer can only start over.
	assert.Equal(t, uint32(0), recoveryPoint(progress, 0))
	assert.Equal(t, int64(0), recoveryOffset(278, 0))
	assert.Equal(t, uint32(0), recoveryPoint(0, negotiated))
}
