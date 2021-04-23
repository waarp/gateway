package sftp

import (
	"context"
	"io"
	"net"
	"path"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

type SSHListener struct {
	DB          *database.DB
	Logger      *log.Logger
	Agent       *model.LocalAgent
	ProtoConfig *config.SftpProtoConfig
	SSHConf     *ssh.ServerConfig
	Listener    net.Listener

	ctx    context.Context
	cancel context.CancelFunc
	connWg sync.WaitGroup

	handlerMaker func(ssh.Channel, *model.LocalAccount) sftp.Handlers
}

func (l *SSHListener) listen() {
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

func (l *SSHListener) handleConnection(parent context.Context, nConn net.Conn) {
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

		var acc model.LocalAccount
		if err := l.DB.Get(&acc, "local_agent_id=? AND login=?", l.Agent.ID,
			servConn.User()).Run(); err != nil {
			l.Logger.Errorf("Failed to retrieve SFTP user: %s", err)
			return
		}

		sesWg := &sync.WaitGroup{}
		for newChannel := range channels {
			l.handleSession(sesWg, &acc, newChannel)
		}
		sesWg.Wait()
	}()
}

func (l *SSHListener) handleSession(wg *sync.WaitGroup, acc *model.LocalAccount,
	newChannel ssh.NewChannel) {

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
		go internal.AcceptRequests(requests)

		server := sftp.NewRequestServer(channel, l.handlerMaker(channel, acc))
		_ = server.Serve()
		_ = server.Close()

		wg.Done()
	}()
}

func (l *SSHListener) makeHandlers(ch ssh.Channel, acc *model.LocalAccount) sftp.Handlers {
	return sftp.Handlers{
		FileGet:  l.makeFileReader(ch, acc),
		FilePut:  l.makeFileWriter(ch, acc),
		FileCmd:  makeFileCmder(),
		FileList: l.makeFileLister(acc),
	}
}

func (l *SSHListener) makeFileReader(ch ssh.Channel, acc *model.LocalAccount) internal.ReaderAtFunc {
	return func(r *sftp.Request) (io.ReaderAt, error) {
		l.Logger.Debug("GET request received")

		// Get rule according to request filepath
		rule, err := internal.GetRuleFromPath(l.DB, r, true)
		if err != nil {
			l.Logger.Errorf("Failed to retrieve transfer rule: %s", err)
			return nil, errDatabase
		}

		// Create Transfer
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  acc.ID,
			LocalPath:  path.Base(r.Filepath),
			RemotePath: path.Base(r.Filepath),
			Start:      time.Now(),
			Status:     types.StatusRunning,
			Step:       types.StepNone,
		}

		l.Logger.Infof("Download of file '%s' requested by '%s' using rule '%s'",
			trans.RemotePath, acc.Login, rule.Name)

		if err := pipeline.NewServerTransfer(l.DB, l.Logger, trans); err != nil {
			return nil, err
		}
		pip, err := pipeline.NewServerPipeline(l.DB, trans, &errorHandler{ch})
		if err != nil {
			return nil, modelToSFTP(err)
		}
		str, err := newStream(pip)
		if err != nil {
			return str, err
		}
		return str, nil
	}
}

func (l *SSHListener) makeFileWriter(ch ssh.Channel, acc *model.LocalAccount) internal.WriterAtFunc {
	return func(r *sftp.Request) (io.WriterAt, error) {
		l.Logger.Debug("PUT request received")

		// Get rule according to request filepath
		rule, err := internal.GetRuleFromPath(l.DB, r, false)
		if err != nil {
			l.Logger.Errorf("Failed to retrieve transfer rule: %s", err)
			return nil, errDatabase
		}

		// Create Transfer
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  acc.ID,
			LocalPath:  path.Base(r.Filepath),
			RemotePath: path.Base(r.Filepath),
			Start:      time.Now(),
			Status:     types.StatusRunning,
			Step:       types.StepNone,
		}

		l.Logger.Infof("Upload of file '%s' requested by '%s' using rule '%s'",
			trans.RemotePath, acc.Login, rule.Name)

		if err := pipeline.NewServerTransfer(l.DB, l.Logger, trans); err != nil {
			return nil, err
		}
		pip, err := pipeline.NewServerPipeline(l.DB, trans, &errorHandler{ch})
		if err != nil {
			return nil, modelToSFTP(err)
		}
		str, err := newStream(pip)
		if err != nil {
			return str, err
		}
		return str, nil
	}
}

func (l *SSHListener) close(ctx context.Context) error {
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
