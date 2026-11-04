package wg

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransfersResumeAll(t *testing.T) {
	t.Run(`Testing the transfer "resume all" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferResumeAll{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   "/api/transfers/resume-all",
			values: url.Values{"rule": {"orders"}, "status": {"ERROR"}, "requested": {"partner"}, "except": {"5"}},
		}

		result := &expectedResponse{status: http.StatusOK, body: map[string]any{
			"done": 2, "refused": 1,
			"refusals": []any{map[string]any{"id": 7, "reason": "state"}},
		}}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--rule", "orders", "--status", "ERROR", "--requested", "partner", "--except", "5"),
					"Then it should not return an error")

				assert.Equal(t,
					"2 transfers resumed, 1 refused.\n"+
						"  - transfer 7: the state of the transfer does not allow it\n",
					w.String(),
					"Then it should display the counted answer, with the reason of the refusal",
				)
			})
		})
	})
}

func TestTransfersRetryAll(t *testing.T) {
	t.Run(`Testing the transfer "retry all" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferRetryAll{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   "/api/transfers/retry-all",
			values: url.Values{"status": {"CANCELLED"}, "requester": {"login"}, "date": {"2031-01-01T00:00:00Z"}},
		}

		result := &expectedResponse{status: http.StatusOK, body: map[string]any{
			"done": 1, "refused": 0, "created": []any{101},
		}}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--status", "CANCELLED", "--requester", "login", "--date", "2031-01-01T00:00:00Z"),
					"Then it should not return an error")

				assert.Equal(t,
					"1 transfer retried, 0 refused.\nNew transfers: 101\n",
					w.String(),
					"Then it should display the counted answer and the new transfer",
				)
			})
		})
	})
}

func TestTransfersCancelAllByFilters(t *testing.T) {
	t.Run(`Testing the transfer "cancel all" command with filters`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferCancelAll{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   "/api/transfers",
			values: url.Values{"rule": {"orders"}, "target": {"error"}},
		}

		result := &expectedResponse{status: http.StatusOK, body: map[string]any{"done": 3, "refused": 0}}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, "--rule", "orders", "--target", "error"),
					"Then it should not return an error")

				assert.Equal(t, "3 transfers canceled, 0 refused.\n", w.String(),
					"Then it should display the counted answer")
			})
		})
	})

	t.Run(`Testing the transfer "cancel all" command with the requested alone`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferCancelAll{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   "/api/transfers",
			values: url.Values{"requested": {"other"}},
		}

		result := &expectedResponse{status: http.StatusOK, body: map[string]any{"done": 1, "refused": 0}}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, "--requested", "other"),
					"Then it should name the set without a target (#843)")

				assert.Equal(t, "1 transfer canceled, 0 refused.\n", w.String(),
					"Then it should display the counted answer")
			})
		})
	})

	t.Run(`Testing the transfer "cancel all" command without any option`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferCancelAll{}

		require.ErrorIs(t, executeCommand(t, w, command), errMissingTarget,
			"Then it should require the target when nothing names the set")
	})
}
