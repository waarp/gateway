package pesit

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

// legacyPreConnection is the 28-byte pre-connection message of the legacy
// profile, under NSDU framing: the length, the 8-byte EBCDIC marker, a blank
// login and password, the 2 extra bytes and a zeroed trailer.
func legacyPreConnection() []byte {
	msg := []byte{0x00, 0x1C, 0xC3, 0xC6, 0xE3, 0xD7, 0xE2, 0xC9, 0xE3, 0xE7}

	for range 16 {
		msg = append(msg, 0x40) // EBCDIC space
	}

	return append(msg, 0x0D, 0x25, 0x00, 0x00)
}

func TestLegacyProfileFollowsCompatibilityMode(t *testing.T) {
	const ackSize = 10 // length, ACK0/NAK0, the 2 extra bytes echoed, trailer

	testCases := []struct {
		mode     string
		accepted bool
	}{
		{CompatibilityModeNonStandard, true},
		{CompatibilityModeStandard, false},
	}

	for _, test := range testCases {
		t.Run(test.mode, func(t *testing.T) {
			ctx := gwtesting.NewTestServerCtx(t, Pesit, map[string]any{
				"compatibilityMode": test.mode,
			})

			conn, err := net.DialTimeout("tcp", ctx.Server.Address.String(), time.Second)
			require.NoError(t, err)

			defer conn.Close()

			require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

			_, err = conn.Write(legacyPreConnection())
			require.NoError(t, err)

			resp := make([]byte, ackSize)
			_, err = io.ReadFull(conn, resp)

			if !test.accepted {
				// A standard server refuses the profile and closes the connection.
				require.Error(t, err)

				return
			}

			require.NoError(t, err, "a non-standard server answers the pre-connection")

			// The blank credentials are refused (NAK0), which is the point: the
			// message was understood and answered in the legacy form.
			assert.Contains(t, [][]byte{{0xC1, 0xC3, 0xD2, 0xF0}, {0xD5, 0xC1, 0xD2, 0xF0}}, resp[2:6])
			assert.Equal(t, []byte{0x0D, 0x25}, resp[6:8], "the 2 extra bytes are echoed")
		})
	}
}
