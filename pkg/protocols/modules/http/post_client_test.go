package http

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// A push refused by the server while the client still writes (#841): the
// transport closes the request body, the write fails, and the goroutine of
// the request delivers the answer of the server a moment later. The error of
// the transfer is the one the server reported, not the write failure.
func TestSendReportsTheRefusalOfTheServer(t *testing.T) {
	t.Parallel()

	body, writer := io.Pipe()
	require.NoError(t, body.Close()) // the transport closed the body: the request has ended

	client := &postClient{
		pip:    &pipeline.Pipeline{Logger: testhelpers.GetTestLogger(t)},
		writer: writer,
		reqErr: make(chan error),
		resp:   make(chan *http.Response),
	}

	refusal := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Request:    &http.Request{Method: http.MethodPost},
		Body:       io.NopCloser(strings.NewReader("")),
		Header: http.Header{
			httpconst.TransferStatus: {string(types.StatusError)},
			httpconst.ErrorCode:      {types.TeExternalOperation.String()},
			httpconst.ErrorMessage:   {"pre-tasks failed"},
		},
	}

	go func() {
		time.Sleep(100 * time.Millisecond) // after the write has failed

		client.resp <- refusal
	}()

	err := client.Send(strings.NewReader("the content of the file"))
	require.NotNil(t, err)
	assert.Equal(t, types.TeExternalOperation, err.Code())
	assert.Equal(t, "Error on remote partner: pre-tasks failed", err.Redacted())
}

// The same, when the request itself failed: the error of the request is
// reported, not the write failure.
func TestSendReportsTheFailureOfTheRequest(t *testing.T) {
	t.Parallel()

	body, writer := io.Pipe()
	require.NoError(t, body.Close())

	client := &postClient{
		pip:    &pipeline.Pipeline{Logger: testhelpers.GetTestLogger(t)},
		writer: writer,
		reqErr: make(chan error),
		resp:   make(chan *http.Response),
	}

	go func() {
		time.Sleep(100 * time.Millisecond)

		client.reqErr <- errors.New("connection reset by peer")
	}()

	err := client.Send(strings.NewReader("the content of the file"))
	require.NotNil(t, err)
	assert.Equal(t, types.TeDataTransfer, err.Code())
	assert.Equal(t, "HTTP transfer failed", err.Redacted())
}

// A write that fails for a reason of its own (the file cannot be read) is
// reported at once: the request is still waiting for the body.
func TestSendReportsItsOwnFailure(t *testing.T) {
	t.Parallel()

	_, writer := io.Pipe()

	client := &postClient{
		pip:    &pipeline.Pipeline{Logger: testhelpers.GetTestLogger(t)},
		writer: writer,
		reqErr: make(chan error),
		resp:   make(chan *http.Response),
	}

	start := time.Now()
	err := client.Send(unreadableFile{})
	require.NotNil(t, err)
	assert.Equal(t, types.TeDataTransfer, err.Code())
	assert.Equal(t, "failed to write to remote HTTP file", err.Redacted())
	assert.Less(t, time.Since(start), time.Second, "reported without waiting for the request")
}

// unreadableFile is a file whose reading fails.
type unreadableFile struct{}

var errUnreadable = errors.New("the file cannot be read")

func (unreadableFile) Read([]byte) (int, error)          { return 0, errUnreadable }
func (unreadableFile) ReadAt([]byte, int64) (int, error) { return 0, errUnreadable }
func (unreadableFile) Seek(int64, int) (int64, error)    { return 0, nil }
