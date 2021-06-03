package pipeline

import (
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func Leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func Branch(s string) utils.Branch { return utils.Branch(s) }

type DataStream interface {
	io.Reader
	io.Writer
	io.ReaderAt
	io.WriterAt
}

type TransferStream interface {
	DataStream
	close() *types.TransferError
	move() *types.TransferError
	stop()
}

func newVoidStream(p *Pipeline) (*voidStream, *types.TransferError) {
	var err error
	if p.TransCtx.Rule.IsSend {
		err = p.machine.Transition("reading")
	} else {
		err = p.machine.Transition("writing")
	}
	if err != nil {
		return nil, types.NewTransferError(types.TeInternal, err.Error())
	}
	return &voidStream{p}, nil
}

type voidStream struct{ *Pipeline }

func (v *voidStream) checkState(state, fun string, defaultN int, defaultErr error) (int, error) {
	if curr := v.machine.Current(); curr != state {
		v.handleStateErr(fun, curr)
		return 0, errStateMachine
	}
	return defaultN, defaultErr
}

func (v *voidStream) Read([]byte) (int, error) {
	return v.checkState("reading", "Read", 0, io.EOF)
}

func (v *voidStream) Write(p []byte) (int, error) {
	return v.checkState("writing", "Write", len(p), nil)
}

func (v *voidStream) ReadAt([]byte, int64) (int, error) {
	return v.checkState("reading", "ReadAt", 0, io.EOF)
}

func (v *voidStream) WriteAt(p []byte, _ int64) (int, error) {
	return v.checkState("writing", "WriteAt", len(p), nil)
}

func (v *voidStream) close() *types.TransferError {
	if err := v.machine.Transition("close"); err != nil {
		v.handleStateErr("close", v.machine.Current())
		return errStateMachine
	}
	return nil
}

func (v *voidStream) move() *types.TransferError {
	if err := v.machine.Transition("move"); err != nil {
		v.handleStateErr("move", v.machine.Current())
		return errStateMachine
	}
	return nil
}

func (*voidStream) stop() {}
