package r66

import (
	"context"
	"fmt"
	"io"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"code.waarp.fr/waarp-r66/r66"
	"code.waarp.fr/waarp-r66/r66/utils"
	"github.com/smartystreets/goconvey/convey"
)

func init() {
	tasks.RunnableTasks["TESTCHECK"] = &testTaskSuccess{}
	tasks.RunnableTasks["TESTFAIL"] = &testTaskFail{}
	model.ValidTasks["TESTCHECK"] = &testTaskSuccess{}
	model.ValidTasks["TESTFAIL"] = &testTaskFail{}

	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

var checkChannel chan string

func shouldFinishOK(cEnd, sEnd string) {
	msg := <-checkChannel
	convey.So(msg, convey.ShouldBeIn, cEnd, sEnd)
	if msg == cEnd {
		convey.So(<-checkChannel, convey.ShouldEqual, sEnd)
	} else {
		convey.So(<-checkChannel, convey.ShouldEqual, cEnd)
	}
	convey.So(checkChannel, convey.ShouldBeEmpty)
}

func shouldFinishError(cErr, cEnd, sErr, sEnd string) {
	msg1 := <-checkChannel
	convey.So(msg1, convey.ShouldBeIn, cErr, sErr)
	if msg1 == cErr {
		msg2 := <-checkChannel
		convey.So(msg2, convey.ShouldBeIn, cEnd, sErr)
		if msg2 == cEnd {
			convey.So(<-checkChannel, convey.ShouldEqual, sErr)
			convey.So(<-checkChannel, convey.ShouldEqual, sEnd)
		} else {
			shouldFinishOK(cEnd, sEnd)
		}
	} else {
		msg2 := <-checkChannel
		convey.So(msg2, convey.ShouldBeIn, cErr, sEnd)
		if msg2 == sEnd {
			convey.So(<-checkChannel, convey.ShouldEqual, cErr)
			convey.So(<-checkChannel, convey.ShouldEqual, cEnd)
		} else {
			shouldFinishOK(cEnd, sEnd)
		}
	}
	convey.So(checkChannel, convey.ShouldBeEmpty)
}

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(map[string]string) error {
	return nil
}
func (t *testTaskSuccess) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	checkChannel <- args["msg"]
	return "", nil
}

type testTaskFail struct {
	msg string
}

func (t *testTaskFail) Validate(map[string]string) error {
	return nil
}

func (t *testTaskFail) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	checkChannel <- args["msg"]
	return "task failed", fmt.Errorf("task failed")
}

func makeDummyServer(c convey.C, handler r66.AuthentHandler) (server *r66.Server, addr string) {
	port := testhelpers.GetFreePort(c)
	addr = fmt.Sprintf("localhost:%d", port)

	server = &r66.Server{
		Login:    "toto",
		Password: []byte("sesame"),
		Conf: r66.Configuration{
			FileSize:   false,
			FinalHash:  false,
			DigestAlgo: "",
			Proxified:  false,
		},
		AuthentHandler: handler,
	}

	go server.ListenAndServe(addr)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c.So(server.Shutdown(ctx), convey.ShouldBeNil)
	})
	return
}

func makeTestHandler() *dummyAuthHandler {
	return &dummyAuthHandler{
		auth: true,
		dummyReqHandler: &dummyReqHandler{
			req: true,
			dummyTransHandler: &dummyTransHandler{
				dummyStream: &dummyStream{file: true},
				pre:         true,
				get:         true,
				end:         true,
				post:        true,
				valid:       true,
			},
		},
	}
}

type dummyAuthHandler struct {
	*dummyReqHandler
	auth bool
}

func (d *dummyAuthHandler) ValidAuth(*r66.Authent) (r66.SessionHandler, error) {
	if d.auth {
		return d.dummyReqHandler, nil
	}
	return nil, &r66.Error{Code: r66.BadAuthent, Detail: "authentication failed"}
}

type dummyReqHandler struct {
	*dummyTransHandler
	req bool
}

func (d *dummyReqHandler) ValidRequest(*r66.Request) (r66.TransferHandler, error) {
	if d.req {
		return d.dummyTransHandler, nil
	}
	return nil, &r66.Error{Code: r66.BadAuthent, Detail: "authentication failed"}
}

type dummyTransHandler struct {
	*dummyStream
	pre, get, end, post, valid bool
}

func (d *dummyTransHandler) RunPreTask() error {
	if d.end {
		return nil
	}
	return &r66.Error{Code: r66.ExternalOperation, Detail: "pre-tasks failed"}
}

func (d *dummyTransHandler) GetStream() (utils.ReadWriterAt, error) {
	if d.end {
		return d.dummyStream, nil
	}
	return nil, &r66.Error{Code: r66.FileNotAllowed, Detail: "failed to open file"}
}

func (d *dummyTransHandler) ValidEndTransfer(*r66.EndTransfer) error {
	if d.end {
		return nil
	}
	return &r66.Error{Code: r66.MD5Error, Detail: "transfer validation failed"}
}

func (d *dummyTransHandler) RunPostTask() error {
	if d.post {
		return nil
	}
	return &r66.Error{Code: r66.ExternalOperation, Detail: "post-tasks failed"}
}

func (d dummyTransHandler) ValidEndRequest() error {
	if d.valid {
		return nil
	}
	return &r66.Error{Code: r66.FinalOp, Detail: "request validation failed"}
}

func (d dummyTransHandler) RunErrorTask(error) error {
	return nil
}

type dummyStream struct{ file bool }

func (d *dummyStream) ReadAt([]byte, int64) (n int, err error) {
	if d.file {
		return 0, io.EOF
	}
	return 0, &r66.Error{Code: r66.Internal, Detail: "file read failed"}
}

func (d *dummyStream) WriteAt(p []byte, _ int64) (n int, err error) {
	if d.file {
		return len(p), nil
	}
	return 0, &r66.Error{Code: r66.Internal, Detail: "file write failed"}
}
