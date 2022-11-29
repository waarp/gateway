package pipeline

import (
	"sync/atomic"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const TransferTimeout = 5 * time.Second

// ErrTestFail is the error returned by the pipeline when TestFailAt is set.
var ErrTestFail = types.NewTransferError(types.TeInternal, "this is an intended test error")

// Tester is a tracer to verify if a transfer is executed as it should. Can be
// initialized with the InitTester function. USE ONLY FOR TESTS.
//
//nolint:gochecknoglobals //only used for tests
var Tester *tester

type errOn uint8

const (
	noError errOn = iota
	FileOpen
	DataRead
	DataWrite
)

type tester struct {
	errOn    errOn
	atOffset int64

	CliPre, ServPre,
	CliData, ServData,
	CliPost, ServPost,
	CliErr, ServErr int32
	CliDone, ServDone chan bool
}

// InitTester initialize the Tester variable to trace a transfer's execution.
// USE ONLY FOR TESTS.
func InitTester(c convey.C) {
	Tester = &tester{}

	Tester.Reset()
	c.Reset(func() { Tester = nil })
}

// AddErrorAt adds a simulated error on the given stage at the given offset.
func (t *tester) AddErrorAt(on errOn, at int64) {
	t.errOn = on
	t.atOffset = at
}

func (t *tester) Reset() {
	if t == nil {
		return
	}

	t.Retry()

	t.CliPre = 0
	t.ServPre = 0
	t.CliData = 0
	t.ServData = 0
	t.CliPost = 0
	t.ServPost = 0
	t.CliErr = 0
	t.ServErr = 0
}

func (t *tester) Retry() {
	if t == nil {
		return
	}

	t.errOn = noError
	t.atOffset = 0

	t.CliDone = make(chan bool)
	t.ServDone = make(chan bool)
}

func (t *tester) getError(curStage errOn, curOffset int64) *types.TransferError {
	if t == nil {
		return nil
	}

	if curStage == t.errOn && curOffset >= t.atOffset {
		return ErrTestFail
	}

	if (t.errOn == DataRead && curStage == DataWrite) ||
		(t.errOn == DataWrite && curStage == DataRead) {
		if curOffset >= t.atOffset {
			const slowDuration = 100 * time.Millisecond

			<-time.NewTimer(slowDuration).C
		}
	}

	return nil
}

func (t *tester) preTasksDone(trans *model.Transfer) {
	if t == nil {
		return
	}

	if trans.IsServer() {
		atomic.AddInt32(&t.ServPre, int32(trans.TaskNumber))
	} else {
		atomic.AddInt32(&t.CliPre, int32(trans.TaskNumber))
	}
}

func (t *tester) dataDone(trans *model.Transfer) {
	if t == nil {
		return
	}

	if trans.IsServer() {
		atomic.CompareAndSwapInt32(&t.ServData, 0, 1)
	} else {
		atomic.CompareAndSwapInt32(&t.CliData, 0, 1)
	}
}

func (t *tester) postTasksDone(trans *model.Transfer) {
	if t == nil {
		return
	}

	if trans.IsServer() {
		atomic.AddInt32(&t.ServPost, int32(trans.TaskNumber))
	} else {
		atomic.AddInt32(&t.CliPost, int32(trans.TaskNumber))
	}
}

func (t *tester) errTasksDone(trans *model.Transfer) {
	if t == nil {
		return
	}

	if trans.IsServer() {
		atomic.AddInt32(&t.ServErr, int32(trans.TaskNumber))
	} else {
		atomic.AddInt32(&t.CliErr, int32(trans.TaskNumber))
	}
}

func (t *tester) done(isServer bool) {
	if t == nil {
		return
	}

	if isServer {
		close(t.ServDone)
	} else {
		close(t.CliDone)
	}
}

func (t *tester) waitDone(stage *int32) {
	const checkInterval = 10 * time.Millisecond

	timer := time.NewTimer(TransferTimeout)
	ticker := time.NewTicker(checkInterval)

	for {
		select {
		case <-timer.C:
			panic("timeout waiting for client transfer end")
		case <-ticker.C:
			if atomic.LoadInt32(stage) != 0 {
				return
			}
		}
	}
}

func (t *tester) WaitClientPreTasks()  { t.waitDone(&t.CliPre) }
func (t *tester) WaitServerPreTasks()  { t.waitDone(&t.ServPre) }
func (t *tester) WaitClientData()      { t.waitDone(&t.CliData) }
func (t *tester) WaitServerData()      { t.waitDone(&t.ServData) }
func (t *tester) WaitClientPostTasks() { t.waitDone(&t.CliPost) }
func (t *tester) WaitServerPostTasks() { t.waitDone(&t.ServPost) }
func (t *tester) WaitClientErrTasks()  { t.waitDone(&t.CliErr) }
func (t *tester) WaitServerErrTasks()  { t.waitDone(&t.ServErr) }

// WaitClientDone waits for the client to have finished its side of the transfer.
func (t *tester) WaitClientDone() {
	timer := time.NewTimer(TransferTimeout)
	select {
	case <-timer.C:
		panic("timeout waiting for client transfer end")
	case <-t.CliDone:
	}
}

// WaitServerDone waits for the server to have finished its side of the transfer.
func (t *tester) WaitServerDone() {
	timer := time.NewTimer(TransferTimeout)
	select {
	case <-timer.C:
		panic("timeout waiting for server transfer end")
	case <-t.ServDone:
	}
}
