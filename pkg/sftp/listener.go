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

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/sftp/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
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

//nolint:funlen // factorizing would add complexity
func (l *sshListener) handleConnection(nConn net.Conn) {
	l.connWg.Add(1)

	go func() {
		defer l.connWg.Done()

		defer func() {
			if err := nConn.Close(); err != nil {
				l.Logger.Warningf("An error occurred while closing the TCP connection: %v", err)
			}
		}()

		servConn, channels, reqs, err := ssh.NewServerConn(nConn, l.SSHConf)
		if err != nil {
			l.Logger.Errorf("Failed to perform handshake: %s", err)

			return
		}

		defer func() {
			if err := servConn.Close(); err != nil {
				l.Logger.Warningf("An error occurred while closing the SFTP connection: %v", err)
			}
		}()

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
					if err := newChannel.Reject(ssh.ResourceShortage, "server shutting down"); err != nil {
						l.Logger.Warningf("An error occurred while rejecting an SFTP channel: %v", err)
					}

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

		if err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type"); err != nil {
			l.Logger.Warningf("An error occurred while rejecting an SFTP channel: %v", err)
		}

		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		l.Logger.Errorf("Failed to accept SFTP session: %s", err)

		return
	}

	go acceptRequests(requests, l.Logger)

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
			if err := server.Close(); err != nil {
				l.Logger.Warningf("An error occurred while closing the SFTP session: %v", err)
			}
		}

		if err := channel.Close(); err != nil {
			l.Logger.Warningf("An error occurred while closing the SFTP channel: %v", err)
		}
	}
	server = sftp.NewRequestServer(channel, l.handlerMaker(endSession, acc))

	if err := server.Serve(); err != nil {
		l.Logger.Warningf("An error occurred while serving SFTP requests: %v", err)
	}

	if err := server.Close(); err != nil {
		l.Logger.Warningf("An error occurred while ending the SFTP session: %v", err)
	}
}

func (l *sshListener) makeHandlers(endSession func(context.Context), acc *model.LocalAccount) sftp.Handlers {
	return sftp.Handlers{
		FileGet:  l.makeFileReader(endSession, acc),
		FilePut:  l.makeFileWriter(endSession, acc),
		FileCmd:  makeFileCmder(),
		FileList: l.makeFileLister(acc),
	}
}

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

		pip, err := newServerPipeline(l.DB, l.Logger, trans, l.runningTransfers, endSession)
		if err != nil {
			return nil, err
		}

		return pip, nil
	}
}

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

		pip, err := newServerPipeline(l.DB, l.Logger, trans, l.runningTransfers, endSession)
		if err != nil {
			return nil, toSFTPErr(err)
		}

		return pip, nil
	}
}

func (l *sshListener) close(ctx context.Context) error {
	if l.shutdown == nil {
		l.shutdown = make(chan struct{})
	}

	close(l.shutdown)

	if err := l.Listener.Close(); err != nil {
		l.Logger.Warningf("An error occurred while closing the network connection: %v", err)
	}

	if err := l.runningTransfers.InterruptAll(ctx); err != nil {
		l.Logger.Errorf("Could not interrupt running transfers")

		return fmt.Errorf("could not interrupt running transfers: %w", err)
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
		return fmt.Errorf("failed interrupt running transfers in time: %w", ctx.Err())
	}
}
