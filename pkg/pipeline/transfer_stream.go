package pipeline

import (
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func branch(s string) utils.Branch { return utils.Branch(s) }

// DataStream is an interface regrouping the common functions used for reading
// and writing data.
type DataStream interface {
	io.Reader
	io.Writer
	io.ReaderAt
	io.WriterAt
}

// TransferStream is an abstraction of the transfer file used by the pipeline.
// It exposes the common functions used for reading and writing data.
type TransferStream interface {
	DataStream
	close() *types.TransferError
	move() *types.TransferError
	stop()
}

func newVoidStream(p *Pipeline) (*voidStream, *types.TransferError) {
	return &voidStream{p}, nil
}

type voidStream struct{ *Pipeline }

func (v *voidStream) checkState(state statemachine.State, fun string, defaultN int,
	defaultErr error) (int, error) {
	if curr := v.machine.Current(); curr != state {
		v.handleStateErr(fun, curr)

		return 0, errStateMachine
	}

	return defaultN, defaultErr
}

func (v *voidStream) Read([]byte) (int, error) {
	return v.checkState(stateReading, "Read", 0, io.EOF)
}

func (v *voidStream) Write(p []byte) (int, error) {
	return v.checkState(stateWriting, "Write", len(p), nil)
}

func (v *voidStream) ReadAt([]byte, int64) (int, error) {
	return v.checkState(stateReading, "ReadAt", 0, io.EOF)
}

func (v *voidStream) WriteAt(p []byte, _ int64) (int, error) {
	return v.checkState(stateWriting, "WriteAt", len(p), nil)
}

func (*voidStream) close() *types.TransferError { return nil }
func (*voidStream) move() *types.TransferError  { return nil }
func (*voidStream) stop()                       {}
