package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sshListener struct {
	Db       *database.Db
	Logger   *log.Logger
	Agent    *model.LocalAgent
	Conf     *ssh.ServerConfig
	Listener net.Listener

	connWg sync.WaitGroup
}

func (l *sshListener) listen() {
	go func() {
		for {
			conn, err := l.Listener.Accept()
			if err != nil {
				break
			}

			l.handleConnection(conn)
		}
		l.connWg.Wait()
	}()
}

func (l *sshListener) handleConnection(nConn net.Conn) {
	l.connWg.Add(1)
	go func() {
		servConn, channels, reqs, err := ssh.NewServerConn(nConn, l.Conf)
		if err != nil {
			l.Logger.Errorf("Failed to perform handshake: %s", err)
			return
		}

		go ssh.DiscardRequests(reqs)

		sesWg := &sync.WaitGroup{}
		for newChannel := range channels {
			accountID, err := getAccountID(l.Db, l.Agent.ID, servConn.User())
			if err != nil {
				l.Logger.Errorf("Failed to retrieve user: %s", err)

			}
			l.handleSession(sesWg, accountID, newChannel)
		}
		sesWg.Wait()
		_ = servConn.Close()
		l.connWg.Done()
	}()
}

func (l *sshListener) handleSession(wg *sync.WaitGroup, accountID uint64, newChannel ssh.NewChannel) {
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

		server := sftp.NewRequestServer(channel, l.makeHandlers(accountID))
		_ = server.Serve()
		_ = server.Close()

		wg.Done()
	}()
}

func (l *sshListener) makeHandlers(accountID uint64) sftp.Handlers {
	root, _ := os.Getwd()
	var conf config.SftpProtoConfig
	if err := json.Unmarshal(l.Agent.ProtoConfig, &conf); err == nil {
		root = conf.Root
	}
	return sftp.Handlers{
		FileGet:  l.makeFileReader(accountID, conf),
		FilePut:  l.makeFileWriter(accountID, conf),
		FileCmd:  nil,
		FileList: makeFileLister(root),
	}
}

func (l *sshListener) makeFileReader(accountID uint64, conf config.SftpProtoConfig) fileReaderFunc {
	return func(r *sftp.Request) (io.ReaderAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := model.Rule{Path: path, IsSend: true}
		if err := l.Db.Get(&rule); err != nil {
			l.Logger.Errorf("No rule found for directory '%s'", path)
			return nil, err
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  accountID,
			SourcePath: filepath.Base(r.Filepath),
			DestPath:   ".",
			Start:      time.Now(),
			Status:     model.StatusPlanned,
		}

		stream, err := newSftpStream(l.Logger, l.Db, conf.Root, trans)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func (l *sshListener) makeFileWriter(accountID uint64, conf config.SftpProtoConfig) fileWriterFunc {
	return func(r *sftp.Request) (io.WriterAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := model.Rule{Path: path, IsSend: false}
		if err := l.Db.Get(&rule); err != nil {
			l.Logger.Errorf("No rule found for directory '%s'", path)
			return nil, err
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    l.Agent.ID,
			AccountID:  accountID,
			SourcePath: ".",
			DestPath:   filepath.Base(r.Filepath),
			Start:      time.Now(),
			Status:     model.StatusPlanned,
		}

		stream, err := newSftpStream(l.Logger, l.Db, conf.Root, trans)
		if err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func (l *sshListener) close(ctx context.Context) error {
	if err := l.Listener.Close(); err != nil {
		return err
	}
	finished := make(chan bool)
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
