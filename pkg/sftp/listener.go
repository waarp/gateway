package sftp

import (
	"context"
	"fmt"
	"io"
	"net"
	"path"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
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

	handlerMaker func(ctx context.Context, accountID uint64) sftp.Handlers
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
				if err := servConn.Wait(); err != nil {
					l.Logger.Warningf("The following error during server stop: %v", err)
				}

				close(closed)
			}()

			select {
			case <-closed:
			case <-ctx.Done():
			}

			if err := servConn.Close(); err != nil {
				l.Logger.Warningf("The following error occurred when the connection was closed: %v", err)
			}
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

			err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			if err != nil {
				l.Logger.Warningf("The following error occurred while we rejected the channel: %v", err)
			}

			return
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			l.Logger.Errorf("Failed to accept SFTP session: %s", err)

			return
		}

		go acceptRequests(requests, l.Logger)

		server := sftp.NewRequestServer(channel, l.handlerMaker(ctx, accountID))
		if err := server.Serve(); err != nil {
			l.Logger.Warningf("The following error occurred while serving sftp requests: %v", err)
		}

		if err := server.Close(); err != nil {
			l.Logger.Warningf("The following error occurred while closing the sftp server for a channel: %v", err)
		}

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
		FileList: l.makeFileLister(paths, accountID),
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

		acc := &model.LocalAccount{}
		if err := l.DB.Get(acc, "id=?", accountID).Run(); err != nil {
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

		stream, err := newSftpStream(ctx, l.Logger, l.DB, paths, &trans)
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

		acc := &model.LocalAccount{}
		if err := l.DB.Get(acc, "id=?", accountID).Run(); err != nil {
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

		stream, err := newSftpStream(ctx, l.Logger, l.DB, paths, &trans)
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
		if err := l.Listener.Close(); err != nil {
			l.Logger.Warningf("An error occurred while closing the network connection: %v", err)
		}

		if err := ctx.Err(); err != nil {
			return fmt.Errorf("the context has been terminated with an error: %w", err)
		}

		return nil
	}
}
