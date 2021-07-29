package sftp

import (
	"context"
	"io"
	"net"
	"path"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
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

	handlerMaker func(ctx context.Context, acc *model.LocalAccount) sftp.Handlers
}

func (l *sshListener) listen() {
	l.handlerMaker = l.makeHandlers
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
			closed := make(chan bool)
			go func() {
				_ = servConn.Wait()
				close(closed)
			}()
			select {
			case <-closed:
			case <-ctx.Done():
			}
			_ = servConn.Close()
		}()

		go ssh.DiscardRequests(reqs)

		sesWg := &sync.WaitGroup{}
		for newChannel := range channels {
			acc, err := getAccount(l.DB, l.Agent.ID, servConn.User())
			if err != nil {
				l.Logger.Errorf("Failed to retrieve user: %s", err)
				continue
			}
			l.handleSession(ctx, sesWg, acc, newChannel)
		}
		sesWg.Wait()
	}()
}

func (l *sshListener) handleSession(ctx context.Context, wg *sync.WaitGroup,
	acc *model.LocalAccount, newChannel ssh.NewChannel) {
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

		server := sftp.NewRequestServer(channel, l.handlerMaker(ctx, acc))
		_ = server.Serve()
		_ = server.Close()

		wg.Done()
	}()
}

func (l *sshListener) makeHandlers(ctx context.Context, acc *model.LocalAccount) sftp.Handlers {
	paths := &pipeline.Paths{
		PathsConfig: l.GWConf.Paths,
		ServerRoot:  l.Agent.Root,
		ServerIn:    l.Agent.InDir,
		ServerOut:   l.Agent.OutDir,
		ServerWork:  l.Agent.WorkDir,
	}

	return sftp.Handlers{
		FileGet:  l.makeFileReader(ctx, acc, paths),
		FilePut:  l.makeFileWriter(ctx, acc, paths),
		FileCmd:  makeFileCmder(),
		FileList: l.makeFileLister(paths, acc),
	}
}

func (l *sshListener) makeFileReader(ctx context.Context, acc *model.LocalAccount,
	paths *pipeline.Paths) fileReaderFunc {
	return func(r *sftp.Request) (io.ReaderAt, error) {
		l.Logger.Info("GET request received")

		// Get rule according to request filepath
		rule, err := l.getClosestRule(acc, r.Filepath, true)
		if err != nil {
			l.Logger.Error(err.Error())
			return nil, err
		}
		if !rule.IsSend {
			return nil, sftp.ErrSSHFxNoSuchFile
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  acc.ID,
			SourceFile: path.Base(r.Filepath),
			DestFile:   path.Base(r.Filepath),
			Start:      time.Now(),
			Status:     types.StatusRunning,
			Step:       types.StepSetup,
		}

		l.Logger.Infof("Download of file '%s' requested by '%s' using rule '%s'",
			trans.SourceFile, acc.Login, rule.Name)

		stream, err := newSftpStream(ctx, l.Logger, l.DB, *paths, trans)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func (l *sshListener) makeFileWriter(ctx context.Context, acc *model.LocalAccount,
	paths *pipeline.Paths) fileWriterFunc {
	return func(r *sftp.Request) (io.WriterAt, error) {
		l.Logger.Debug("PUT request received")

		// Get rule according to request filepath
		rule, err := l.getClosestRule(acc, r.Filepath, false)
		if err != nil {
			l.Logger.Error(err.Error())
			return nil, err
		}
		if rule.IsSend {
			return nil, sftp.ErrSSHFxPermissionDenied
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  acc.ID,
			SourceFile: path.Base(r.Filepath),
			DestFile:   path.Base(r.Filepath),
			Start:      time.Now(),
			Status:     types.StatusRunning,
			Step:       types.StepSetup,
		}

		l.Logger.Infof("Upload of file '%s' requested by '%s' using rule '%s'",
			trans.SourceFile, acc.Login, rule.Name)

		stream, err := newSftpStream(ctx, l.Logger, l.DB, *paths, trans)
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
		_ = l.Listener.Close()
		return ctx.Err()
	}
}
