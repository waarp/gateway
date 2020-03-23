package sftp

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sshListener struct {
	DB          *database.DB
	Logger      *log.Logger
	Agent       *model.LocalAgent
	ProtoConfig *config.SftpProtoConfig
	GWConf      *conf.ServerConfig
	SSHConf     *ssh.ServerConfig
	Listener    net.Listener

	ctx    context.Context
	cancel context.CancelFunc
	connWg sync.WaitGroup
}

func (l *sshListener) listen() {
	go func() {
		for {
			conn, err := l.Listener.Accept()
			if err != nil {
				break
			}

			l.handleConnection(l.ctx, conn)
		}
	}()
}

func (l *sshListener) handleConnection(parent context.Context, nConn net.Conn) {
	l.connWg.Add(1)
	ctx, cancel := context.WithCancel(parent)

	go func() {
		defer cancel()
		defer l.connWg.Done()

		servConn, channels, reqs, err := ssh.NewServerConn(nConn, l.SSHConf)
		if err != nil {
			l.Logger.Errorf("Failed to perform handshake: %s", err)
			return
		}
		go func() {
			<-ctx.Done()
			_ = servConn.Close()
		}()

		go ssh.DiscardRequests(reqs)

		sesWg := &sync.WaitGroup{}
		for newChannel := range channels {
			accountID, err := getAccountID(l.DB, l.Agent.ID, servConn.User())
			if err != nil {
				l.Logger.Errorf("Failed to retrieve user: %s", err)
				continue
			}
			l.handleSession(ctx, sesWg, accountID, newChannel)
		}
		sesWg.Wait()
	}()
}

func (l *sshListener) handleSession(ctx context.Context, wg *sync.WaitGroup,
	accountID uint64, newChannel ssh.NewChannel) {

	wg.Add(1)
	go func() {
		if newChannel.ChannelType() != "session" {
			l.Logger.Warning("Unknown channel type received")
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			return
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			l.Logger.Errorf("Failed to accept SFTP session: %s", err)
			return
		}
		go acceptRequests(requests)

		server := sftp.NewRequestServer(channel, l.makeHandlers(ctx, accountID))
		_ = server.Serve()
		_ = server.Close()

		wg.Done()
	}()
}

func (l *sshListener) makeHandlers(ctx context.Context, accountID uint64) sftp.Handlers {

	return sftp.Handlers{
		FileGet:  l.makeFileReader(ctx, accountID),
		FilePut:  l.makeFileWriter(ctx, accountID),
		FileCmd:  makeFileCmder(),
		FileList: makeFileLister(l.Agent.Root),
	}
}

func (l *sshListener) makeFileReader(ctx context.Context, accountID uint64) fileReaderFunc {

	return func(r *sftp.Request) (io.ReaderAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := model.Rule{Path: path, IsSend: true}
		if err := l.DB.Get(&rule); err != nil {
			l.Logger.Errorf("No rule found for directory '%s'", path)
			return nil, fmt.Errorf("cannot retrieve transfer rule: %s", err)
		}
		paths := pipeline.Paths{
			PathsConfig: l.GWConf.Paths,
			ServerRoot:  l.Agent.Root,
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  accountID,
			SourceFile: filepath.Base(r.Filepath),
			DestFile:   ".",
			Start:      time.Now(),
			Status:     model.StatusRunning,
			Step:       model.StepSetup,
		}

		stream, err := newSftpStream(ctx, l.Logger, l.DB, paths, trans)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func (l *sshListener) makeFileWriter(ctx context.Context, accountID uint64) fileWriterFunc {

	return func(r *sftp.Request) (io.WriterAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := model.Rule{Path: path, IsSend: false}
		if err := l.DB.Get(&rule); err != nil {
			l.Logger.Errorf("No rule found for directory '%s'", path)
			return nil, fmt.Errorf("cannot retrieve transfer rule: %s", err)
		}
		paths := pipeline.Paths{
			PathsConfig: l.GWConf.Paths,
			ServerRoot:  l.Agent.Root,
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  accountID,
			SourceFile: ".",
			DestFile:   filepath.Base(r.Filepath),
			Start:      time.Now(),
			Status:     model.StatusRunning,
			Step:       model.StepSetup,
		}

		stream, err := newSftpStream(ctx, l.Logger, l.DB, paths, trans)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func (l *sshListener) close(ctx context.Context) error {
	l.cancel()

	finished := make(chan error)
	go func() {
		l.connWg.Wait()
		close(finished)
	}()

	select {
	case err := <-finished:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
