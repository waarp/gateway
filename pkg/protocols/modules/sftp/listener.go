package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type sshListener struct {
	DB       *database.DB
	Logger   *log.Logger
	Server   *model.LocalAgent
	SSHConf  *ssh.ServerConfig
	Listener net.Listener

	tracer   func() pipeline.Trace
	shutdown chan struct{}

	handlerMaker func(*model.LocalAccount) sftp.Handlers
}

func (l *sshListener) listen() {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			select {
			case <-l.shutdown:
				return
			default:
				l.Logger.Error("Failed to accept connection: %s", err)
			}

			continue
		}

		go l.handleConnection(conn)
	}
}

//nolint:funlen // factorizing would add complexity
func (l *sshListener) handleConnection(nConn net.Conn) {
	analytics.AddConnection()
	defer analytics.SubConnection()

	defer closeTCPConn(nConn, l.Logger)

	servConn, channels, reqs, err := ssh.NewServerConn(nConn, l.SSHConf)
	if err != nil {
		l.Logger.Error("Failed to perform handshake: %s", err)

		return
	}

	defer closeSSHConn(servConn, l.Logger)

	go ssh.DiscardRequests(reqs)

	var acc model.LocalAccount
	if err := l.DB.Get(&acc, "local_agent_id=? AND login=?", l.Server.ID,
		servConn.User()).Run(); err != nil {
		l.Logger.Error("Failed to retrieve SFTP user: %s", err)

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
					l.Logger.Warning("An error occurred while rejecting an SFTP channel: %v", err)
				}

			default:
				sesWg.Add(1)

				go l.handleSession(sesWg, &acc, newChannel)
			}
		}
	}
}

func (l *sshListener) handleSession(sesWg *sync.WaitGroup,
	acc *model.LocalAccount, newChannel ssh.NewChannel,
) {
	defer sesWg.Done()

	if newChannel.ChannelType() != "session" {
		l.Logger.Warning("Unknown channel type received")

		if err := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type"); err != nil {
			l.Logger.Warning("An error occurred while rejecting an SFTP channel: %v", err)
		}

		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		l.Logger.Error("Failed to accept SFTP session: %s", err)

		return
	}

	go acceptRequests(requests, l.Logger)

	server := sftp.NewRequestServer(channel, l.handlerMaker(acc))

	if err := server.Serve(); err != nil && !errors.Is(err, io.EOF) {
		l.Logger.Warning("An error occurred while serving SFTP requests: %v", err)
	}

	if err := server.Close(); err != nil {
		l.Logger.Warning("An error occurred while ending the SFTP session: %v", err)
	}
}

func (l *sshListener) makeHandlers(acc *model.LocalAccount) sftp.Handlers {
	return sftp.Handlers{
		FileGet:  l.makeFileReader(acc),
		FilePut:  l.makeFileWriter(acc),
		FileCmd:  l.makeFileCmder(acc),
		FileList: l.makeFileLister(acc),
	}
}

func (l *sshListener) makeFileReader(acc *model.LocalAccount) internal.ReaderAtFunc {
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

		filePath := strings.TrimPrefix(r.Filepath, "/")
		filePath = strings.TrimPrefix(filePath, rule.Path)
		filePath = strings.TrimPrefix(filePath, "/")

		// Create Transfer
		trans := &model.Transfer{
			RuleID:         rule.ID,
			LocalAccountID: utils.NewNullInt64(acc.ID),
			SrcFilename:    filePath,
			Filesize:       model.UnknownSize,
			Start:          time.Now(),
			Status:         types.StatusRunning,
			Step:           types.StepNone,
		}

		l.Logger.Info("Download of file '%s' requested by '%s' using rule '%s'",
			filePath, acc.Login, rule.Name)

		pip, err := newServerPipeline(l.DB, l.Logger, trans, l.tracer)
		if err != nil {
			return nil, err
		}

		return pip, nil
	}
}

func (l *sshListener) makeFileWriter(acc *model.LocalAccount) internal.WriterAtFunc {
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

		filePath := strings.TrimPrefix(r.Filepath, "/")
		filePath = strings.TrimPrefix(filePath, rule.Path)
		filePath = strings.TrimPrefix(filePath, "/")

		// Create Transfer
		trans := &model.Transfer{
			RuleID:         rule.ID,
			LocalAccountID: utils.NewNullInt64(acc.ID),
			DestFilename:   filePath,
			Filesize:       model.UnknownSize,
			Start:          time.Now(),
			Status:         types.StatusRunning,
			Step:           types.StepNone,
		}

		l.Logger.Info("Upload of file '%s' requested by '%s' using rule '%s'",
			filePath, acc.Login, rule.Name)

		pip, err := newServerPipeline(l.DB, l.Logger, trans, l.tracer)
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

	defer func() {
		if err := l.Listener.Close(); err != nil {
			l.Logger.Warning("An error occurred while closing the network connection: %v", err)
		}
	}()

	if err := pipeline.List.StopAllFromServer(ctx, l.Server.ID); err != nil {
		l.Logger.Error("Failed to stop the ongoing SFTP transfers: %v", err)

		return fmt.Errorf("failed to stop the ongoing SFTP transfers: %w", err)
	}

	return nil
}
