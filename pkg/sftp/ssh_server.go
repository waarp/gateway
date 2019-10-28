package sftp

import (
	"context"
	"net"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const maxConn = 5

type result struct {
	*model.Transfer
	error
}

type sshServer struct {
	db       *database.Db
	logger   *log.Logger
	listener net.Listener
	agent    *model.LocalAgent
	conf     *ssh.ServerConfig

	connections chan net.Conn
	results     chan result
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
		results:     make(chan result),
		finished:    make(chan bool),
		shutdown:    make(chan bool),
	}
}

func (s *sshServer) handle(wg *sync.WaitGroup) {
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

		for newChannel := range channels {
			select {
			case <-s.shutdown:
				_ = newChannel.Reject(ssh.ResourceShortage, "server shutting down")
				continue
			default:
			}

			if newChannel.ChannelType() != "session" {
				s.logger.Warning("Unknown channel type received")
				_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}

			channel, requests, err := newChannel.Accept()
			if err != nil {
				s.logger.Errorf("Failed to accept SFTP session: %s", err)
				continue
			}
			go acceptRequests(requests)

			// Resolve conn.User() to model.LocalAccount
			acc := &model.LocalAccount{
				LocalAgentID: s.agent.ID,
				Login:        servConn.Conn.User(),
			}
			if err := s.db.Get(acc); err != nil {
				s.logger.Errorf("Failed to retrieve user: %s", err)
				continue
			}

			report := make(chan *model.Transfer, 1)
			server := sftp.NewRequestServer(channel, makeHandlers(s.db, s.agent,
				acc, report))

			errReport := make(chan error, 1)
			go s.handleAnswer(server, report, errReport)
			errReport <- server.Serve()
			close(errReport)
			_ = channel.Close()
		}
		_ = servConn.Close()
	}
}

func (s *sshServer) listen() {

	wg := &sync.WaitGroup{}
	for i := 0; i < maxConn; i++ {
		wg.Add(1)
		go s.handle(wg)
	}

	go func() {
		wg.Wait()
		close(s.results)
	}()

	go s.toHistory()

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
