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
	paths := &pipeline.Paths{
		PathsConfig: l.GWConf.Paths,
		ServerRoot:  l.Agent.Root,
		ServerIn:    l.Agent.InDir,
		ServerOut:   l.Agent.OutDir,
		ServerWork:  l.Agent.WorkDir,
	}

	return sftp.Handlers{
		FileGet:  l.makeFileReader(ctx, accountID, paths),
		FilePut:  l.makeFileWriter(ctx, accountID, paths),
		FileCmd:  makeFileCmder(),
		FileList: l.makeFileLister(paths),
	}
}

func (l *sshListener) makeFileReader(ctx context.Context, accountID uint64,
	paths *pipeline.Paths) fileReaderFunc {
	return func(r *sftp.Request) (io.ReaderAt, error) {
		l.Logger.Info("GET request received")

		// Get rule according to request filepath
		rule, err := getRuleFromPath(l.DB, r, true)
		if err != nil {
			l.Logger.Error(err.Error())
			return nil, err
		}

		acc := &model.LocalAccount{ID: accountID}
		if err := l.DB.Get(acc); err != nil {
			l.Logger.Error(err.Error())
			return nil, err
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  accountID,
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

func (l *sshListener) makeFileWriter(ctx context.Context, accountID uint64,
	paths *pipeline.Paths) fileWriterFunc {
	return func(r *sftp.Request) (io.WriterAt, error) {
		l.Logger.Debug("PUT request received")

		acc := &model.LocalAccount{ID: accountID}
		if err := l.DB.Get(acc); err != nil {
			l.Logger.Error(err.Error())
			return nil, err
		}

		// Get rule according to request filepath
		rule, err := getRuleFromPath(l.DB, r, false)
		if err != nil {
			l.Logger.Error(err.Error())
			return nil, err
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  accountID,
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
		return ctx.Err()
	}
}
