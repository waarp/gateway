package wg

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The choices of the --ack flag are the cases of the model, in their order,
// on the transfers list and on the history list alike (#873): the flag cannot
// read them at run time, a struct tag being a literal.
func TestAckFlagChoices(t *testing.T) {
	t.Parallel()

	want := make([]string, 0, len(model.AckCases()))
	for _, c := range model.AckCases() {
		want = append(want, string(c))
	}

	choice := regexp.MustCompile(`choice:"([^"]+)"`)

	for _, command := range []any{TransferList{}, HistoryList{}} {
		field, ok := reflect.TypeOf(command).FieldByName("Ack")
		require.Truef(t, ok, "%T has no Ack flag", command)
		assert.Equal(t, "ack", field.Tag.Get("long"))

		got := make([]string, 0, len(want))
		for _, m := range choice.FindAllStringSubmatch(string(field.Tag), -1) {
			got = append(got, m[1])
		}

		assert.Equalf(t, want, got, "the choices of --ack on %T", command)
	}
}
