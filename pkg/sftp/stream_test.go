package sftp

import (
	"io/ioutil"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewStream(t *testing.T) {
	Convey("Given database suited for transfers", t, func(c C) {
		ctx := testhelpers.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		initRcvStream(c, ctx)

		Convey("When creating a new stream", func(c C) {
			var server testServer = make(chan struct{})
			pip, err := pipeline.NewServerPipeline(ctx.DB, ctx.Trans, server)
			So(err, ShouldBeNil)
			list := &SSHListener{runningTransfers: pipeline.NewTransferMap()}
			str, err := list.newStream(pip, ctx.Trans)
			So(err, ShouldBeNil)
			Reset(func() { _ = str.pipeline.EndData() })

			Convey("Then it should have created a pipeline", func(c C) {
				So(str.pipeline, ShouldNotBeNil)

				Convey("Then it should have executed the pre-tasks", func(c C) {
					testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")

					Convey("Then it should have opened the file", func(c C) {
						So(str.file, ShouldNotBeNil)
					})
				})
			})
		})
	})
}

func TestStreamReadAt(t *testing.T) {
	Convey("Given database suited for transfers", t, func(c C) {
		ctx := testhelpers.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		initSndStream(c, ctx)

		Convey("Given an SFTP stream", func(c C) {
			var server testServer = make(chan struct{})
			pip, err := pipeline.NewServerPipeline(ctx.DB, ctx.Trans, server)
			So(err, ShouldBeNil)
			list := &SSHListener{runningTransfers: pipeline.NewTransferMap()}
			str, err := list.newStream(pip, ctx.Trans)
			So(err, ShouldBeNil)
			Reset(func() { _ = str.pipeline.EndData() })

			Convey("When reading the stream", func(c C) {
				p := make([]byte, 4)
				_, err := str.ReadAt(p, 0)
				So(err, ShouldBeNil)

				Convey("Then it should have read the file", func(c C) {
					So(p, ShouldResemble, testFileContent[:4])
				})
			})
		})
	})
}

func TestStreamWriteAt(t *testing.T) {
	Convey("Given database suited for transfers", t, func(c C) {
		ctx := testhelpers.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		initRcvStream(c, ctx)

		Convey("Given an SFTP stream", func(c C) {
			var server testServer = make(chan struct{})
			pip, err := pipeline.NewServerPipeline(ctx.DB, ctx.Trans, server)
			So(err, ShouldBeNil)
			list := &SSHListener{runningTransfers: pipeline.NewTransferMap()}
			str, err := list.newStream(pip, ctx.Trans)
			So(err, ShouldBeNil)
			Reset(func() { _ = str.pipeline.EndData() })

			Convey("When reading the stream", func(c C) {
				p := []byte("content")
				_, err := str.WriteAt(p, 0)
				So(err, ShouldBeNil)

				Convey("Then it should have written the file", func(c C) {
					content, err := ioutil.ReadFile(ctx.Trans.LocalPath)
					So(err, ShouldBeNil)
					So(content, ShouldResemble, p)
				})
			})
		})
	})
}

func TestStreamClose(t *testing.T) {
	Convey("Given database suited for transfers", t, func(c C) {
		ctx := testhelpers.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		initRcvStream(c, ctx)

		Convey("Given an SFTP stream", func(c C) {
			var server testServer = make(chan struct{})
			pip, err := pipeline.NewServerPipeline(ctx.DB, ctx.Trans, server)
			So(err, ShouldBeNil)
			list := &SSHListener{runningTransfers: pipeline.NewTransferMap()}
			str, err := list.newStream(pip, ctx.Trans)
			So(err, ShouldBeNil)

			Convey("When closing the stream", func(c C) {
				So(str.Close(), ShouldBeNil)

				Convey("Then it should have executed the post-tasks", func(c C) {
					<-testhelpers.ServerCheckChannel
					testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
				})
			})
		})
	})
}
