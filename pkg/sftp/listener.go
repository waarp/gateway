package sftp

import (
	"context"
	"io"
	"net"
	"path"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sshListener struct {
	DB          *database.DB
	Logger      *log.Logger
	Agent       *model.LocalAgent
	ProtoConfig *config.SftpProtoConfig
	SSHConf     *ssh.ServerConfig
	Listener    net.Listener

	connWg   sync.WaitGroup
	shutdown chan struct{}

	handlerMaker     func(func(context.Context), *model.LocalAccount) sftp.Handlers
	runningTransfers *service.TransferMap
}

func (l *sshListener) listen() {
	l.handlerMaker = l.makeHandlers
	go func() {
		for {
			conn, err := l.Listener.Accept()
			if err != nil {
				select {
				case <-l.shutdown:
					return
				default:
					l.Logger.Errorf("Failed to accept connection: %s", err)
				}
				continue
			}

			l.handleConnection(conn)
		}
	}()
}

func (l *sshListener) handleConnection(nConn net.Conn) {
	l.connWg.Add(1)

	go func() {
		defer l.connWg.Done()
		defer func() { _ = nConn.Close() }()

		servConn, channels, reqs, err := ssh.NewServerConn(nConn, l.SSHConf)
		if err != nil {
			l.Logger.Errorf("Failed to perform handshake: %s", err)
			return
		}
		defer func() { _ = servConn.Close() }()
		go ssh.DiscardRequests(reqs)

		var acc model.LocalAccount
		if err := l.DB.Get(&acc, "local_agent_id=? AND login=?", l.Agent.ID,
			servConn.User()).Run(); err != nil {
			l.Logger.Errorf("Failed to retrieve SFTP user: %s", err)
			return
		}

		sesWg := &sync.WaitGroup{}
		defer sesWg.Wait()
		for {
			select {
			case <-l.shutdown:
				return
			case newChannel, ok := <-channels:
				if !ok {
					return
				}
				select {
				case <-l.shutdown:
					_ = newChannel.Reject(ssh.ResourceShortage, "server shutting down")
				default:
					sesWg.Add(1)
					go func() {
						defer sesWg.Done()
						l.handleSession(&acc, newChannel)
					}()
				}
			}
		}
	}()
}

func (l *sshListener) handleSession(acc *model.LocalAccount, newChannel ssh.NewChannel) {

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

	done := make(chan struct{})
	defer close(done)
	var server *sftp.RequestServer
	endSession := func(ctx context.Context) {
		timer := time.NewTimer(time.Second)
		select {
		case <-done:
		case <-timer.C:
		case <-ctx.Done():
		}
		if server != nil {
			_ = server.Close()
		}
		_ = channel.Close()
	}
	server = sftp.NewRequestServer(channel, l.handlerMaker(endSession, acc))
	_ = server.Serve()
	_ = server.Close()
}

func (l *sshListener) makeHandlers(endSession func(context.Context), acc *model.LocalAccount) sftp.Handlers {
	return sftp.Handlers{
		FileGet:  l.makeFileReader(endSession, acc),
		FilePut:  l.makeFileWriter(endSession, acc),
		FileCmd:  makeFileCmder(),
		FileList: l.makeFileLister(acc),
	}
}

//nolint:dupl
func (l *sshListener) makeFileReader(endSession func(context.Context), acc *model.LocalAccount,
) internal.ReaderAtFunc {
	return func(r *sftp.Request) (io.ReaderAt, error) {
		l.Logger.Debug("GET request received")

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
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  acc.ID,
			LocalPath:  path.Base(r.Filepath),
			RemotePath: path.Base(r.Filepath),
			Filesize:   model.UnknownSize,
			Start:      time.Now(),
			Status:     types.StatusRunning,
			Step:       types.StepNone,
		}

		l.Logger.Infof("Download of file '%s' requested by '%s' using rule '%s'",
			trans.RemotePath, acc.Login, rule.Name)

		pip, err := internal.NewServerPipeline(l.DB, l.Logger, trans, l.runningTransfers, endSession)
		if err != nil {
			return nil, err
		}
		return pip, nil
	}
}

//nolint:dupl
func (l *sshListener) makeFileWriter(endSession func(context.Context), acc *model.LocalAccount,
) internal.WriterAtFunc {
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
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  acc.ID,
			LocalPath:  path.Base(r.Filepath),
			RemotePath: path.Base(r.Filepath),
			Filesize:   model.UnknownSize,
			Start:      time.Now(),
			Status:     types.StatusRunning,
			Step:       types.StepNone,
		}

		l.Logger.Infof("Upload of file '%s' requested by '%s' using rule '%s'",
			trans.RemotePath, acc.Login, rule.Name)

		pip, err := internal.NewServerPipeline(l.DB, l.Logger, trans, l.runningTransfers, endSession)
		if err != nil {
			return nil, err
		}
		return pip, nil
	}
}

func (l *sshListener) close(ctx context.Context) error {
	if l.shutdown == nil {
		l.shutdown = make(chan struct{})
	}
	close(l.shutdown)

	_ = l.Listener.Close()
	if err := l.runningTransfers.InterruptAll(ctx); err != nil {
		l.Logger.Errorf("Could not interrupt running transfers")
		return err
	}
	finished := make(chan struct{})
	go func() {
		l.connWg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
