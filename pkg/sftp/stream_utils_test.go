package sftp

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type testSFTPStream struct {
	c C
	*stream
}

func (t *testSFTPStream) Close() error {
	err := t.stream.Close()
	pipeline.WaitEndTransfer(t.c, t.stream.pipeline)
	testhelpers.ServerCheckChannel <- "SERVER TRANSFER END"
	return err
}

func (l *SSHListener) makeTestFileReader(c C, ch ssh.Channel, acc *model.LocalAccount) internal.ReaderAtFunc {

	handler := l.makeFileReader(ch, acc)
	return func(r *sftp.Request) (io.ReaderAt, error) {
		reader, err := handler(r)
		if err != nil {
			if str, ok := reader.(*stream); ok && str != nil {
				pipeline.WaitEndTransfer(c, str.pipeline)
			}
			testhelpers.ServerCheckChannel <- "SERVER TRANSFER END"
			return nil, err
		}
		return &testSFTPStream{c, reader.(*stream)}, nil
	}
}

func (l *SSHListener) makeTestFileWriter(c C, ch ssh.Channel, acc *model.LocalAccount) internal.WriterAtFunc {

	handler := l.makeFileWriter(ch, acc)
	return func(r *sftp.Request) (io.WriterAt, error) {
		writer, err := handler(r)
		if err != nil {
			if str, ok := writer.(*stream); ok && str != nil {
				pipeline.WaitEndTransfer(c, str.pipeline)
			}
			testhelpers.ServerCheckChannel <- "SERVER TRANSFER END"
			return nil, err
		}
		return &testSFTPStream{c, writer.(*stream)}, nil
	}
}

func (l *SSHListener) makeTestHandlers(c C) func(ssh.Channel, *model.LocalAccount) sftp.Handlers {
	return func(ch ssh.Channel, acc *model.LocalAccount) sftp.Handlers {
		return sftp.Handlers{
			FileGet:  l.makeTestFileReader(c, ch, acc),
			FilePut:  l.makeTestFileWriter(c, ch, acc),
			FileCmd:  makeFileCmder(),
			FileList: l.makeFileLister(acc),
		}
	}
}

func initRcvStream(c C, ctx *testhelpers.Context) {
	trans := &model.Transfer{
		RuleID:     ctx.ServerPush.ID,
		IsServer:   false,
		AgentID:    ctx.Server.ID,
		AccountID:  ctx.LocAccount.ID,
		LocalPath:  "transfer_file",
		RemotePath: "transfer_file",
		Start:      time.Now(),
	}
	c.So(pipeline.NewServerTransfer(ctx.DB, ctx.Logger, trans), ShouldBeNil)

	testhelpers.ClientCheckChannel = make(chan string, 10)
	testhelpers.ServerCheckChannel = make(chan string, 10)

	ctx.Trans = trans
}

var testFileContent = []byte("file content")

func initSndStream(c C, ctx *testhelpers.Context) {
	testFile := filepath.Join(ctx.Paths.GatewayHome, ctx.Paths.DefaultOutDir,
		"transfer_file")
	c.So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

	trans := &model.Transfer{
		RuleID:     ctx.ServerPull.ID,
		IsServer:   false,
		AgentID:    ctx.Server.ID,
		AccountID:  ctx.LocAccount.ID,
		LocalPath:  "transfer_file",
		RemotePath: "transfer_file",
		Start:      time.Now(),
	}
	c.So(pipeline.NewServerTransfer(ctx.DB, ctx.Logger, trans), ShouldBeNil)

	testhelpers.ClientCheckChannel = make(chan string, 10)
	testhelpers.ServerCheckChannel = make(chan string, 10)

	ctx.Trans = trans
}
