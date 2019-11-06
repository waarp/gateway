package sftp

import (
	"context"
	"net"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"golang.org/x/crypto/ssh"
)

const maxConn = 5

type sshServer struct {
	db       *database.Db
	logger   *log.Logger
	listener net.Listener
	agent    *model.LocalAgent
	conf     *ssh.ServerConfig

	connections chan net.Conn
	finished    chan bool
	shutdown    chan bool
}

func newListener(db *database.Db, logger *log.Logger, listener net.Listener,
	agent *model.LocalAgent, conf *ssh.ServerConfig) *sshServer {
	return &sshServer{
		db:          db,
		logger:      logger,
		listener:    listener,
		agent:       agent,
		conf:        conf,
		connections: make(chan net.Conn),
		finished:    make(chan bool),
		shutdown:    make(chan bool),
	}
}

func (s *sshServer) handleConnection(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	for conn := range s.connections {
		servConn, channels, reqs, err := ssh.NewServerConn(conn, s.conf)
		if err != nil {
			s.logger.Errorf("Failed to perform handshake: %s", err)
			continue
		}

		go ssh.DiscardRequests(reqs)

	sessionLoop:
		for {
			var newChannel ssh.NewChannel
			var ok bool
			select {
			case newChannel, ok = <-channels:
				if !ok {
					break sessionLoop
				}
				select {
				case <-s.shutdown:
					_ = newChannel.Reject(ssh.ResourceShortage, "server shutting down")
					continue
				default:
				}
			case <-s.shutdown:
				_ = servConn.Close()
				continue
			}

			s.handleSession(servConn.User(), newChannel)
		}
		_ = servConn.Close()
	}
}

func (s *sshServer) listen() {

	wg := &sync.WaitGroup{}
	for i := 0; i < maxConn; i++ {
		wg.Add(1)
		go s.handleConnection(wg)
	}

	go func() {
		wg.Wait()
		close(s.finished)
	}()

	go func() {
		for {
			nConn, err := s.listener.Accept()
			if err != nil {
				close(s.connections)
				return
			}

			s.connections <- nConn
		}
	}()
}

func (s *sshServer) close(ctx context.Context) error {
	close(s.shutdown)
	_ = s.listener.Close()

	select {
	case <-s.finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
