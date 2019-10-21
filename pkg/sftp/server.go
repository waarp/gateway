package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

// Server represents an instance of SFTP server.
type Server struct {
	Db     *database.Db
	Config *model.LocalAgent

	shutdown chan bool
	logger   *log.Logger
	state    service.State
}

func loadCert(db *database.Db, server *model.LocalAgent) (*model.Cert, error) {
	cert := &model.Cert{OwnerType: server.TableName(), OwnerID: server.ID}
	if err := db.Get(cert); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("no certificate found for SFTP server '%s'",
				server.Name)
		}
		return nil, err
	}

	return cert, nil
}

func loadSSHConfig(db *database.Db, cert *model.Cert) (*ssh.ServerConfig, error) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			user := &model.LocalAccount{Login: c.User()}
			if err := db.Get(user); err != nil {
				return nil, err
			}
			if bcrypt.CompareHashAndPassword(user.Password, pass) != nil {
				return nil, fmt.Errorf("authentication failed")
			}

			return &ssh.Permissions{}, nil
		},
	}

	privateKey, err := ssh.ParsePrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, err
	}
	config.AddHostKey(privateKey)

	return config, nil
}

func parseServerAddr(server *model.LocalAgent) (string, uint16, error) {
	conf := map[string]interface{}{}

	if err := json.Unmarshal(server.ProtoConfig, &conf); err != nil {
		return "", 0, err
	}

	a := conf["address"]
	p := conf["port"]

	addr, ok := a.(string)
	if !ok {
		return "", 0, fmt.Errorf("invalid SFTP server address")
	}

	port, ok := p.(float64)
	if !ok {
		return "", 0, fmt.Errorf("invalid SFTP server port")
	}

	return addr, uint16(port), nil
}

func (s *Server) listen(listener net.Listener, config *ssh.ServerConfig) {
	var server *sftp.RequestServer
	shutdown := make(chan bool)

	go func() {
		<-s.shutdown
		_ = listener.Close()
		if server != nil {
			_ = server.Close()
		}
		shutdown <- true
	}()
	for {
		nConn, err := listener.Accept()
		if err != nil {
			break
		}

		conn, channels, reqs, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			continue
		}

		// Resolve conn.User() to model.LocalAccount
		acc := &model.LocalAccount{
			LocalAgentID: s.Config.ID,
			Login:        conn.Conn.User(),
		}
		if err := s.Db.Get(acc); err != nil {
			continue
		}

		go ssh.DiscardRequests(reqs)

		for newChannel := range channels {
			if newChannel.ChannelType() != "session" {
				_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}

			channel, err := acceptChannel(newChannel)
			if err != nil {
				break
			}

			server = sftp.NewRequestServer(channel, makeHandlers(s.Db, s.Config, acc, shutdown))
			if err := server.Serve(); err != nil && err != io.EOF {
				break
			}

			_ = server.Close()
			_ = nConn.Close()
		}
	}
}

func acceptChannel(newChannel ssh.NewChannel) (ssh.Channel, error) {
	channel, requests, err := newChannel.Accept()
	if err != nil {
		return nil, err
	}

	go func(in <-chan *ssh.Request) {
		for req := range in {
			ok := false
			switch req.Type {
			case "subsystem":
				if string(req.Payload[4:]) == "sftp" {
					ok = true
				}
			}
			_ = req.Reply(ok, nil)
		}
	}(requests)
	return channel, nil
}

// Start starts the SFTP server.
func (s *Server) Start() error {
	start := func() error {
		cert, err := loadCert(s.Db, s.Config)
		if err != nil {
			return err
		}

		sshConf, err := loadSSHConfig(s.Db, cert)
		if err != nil {
			return err
		}

		addr, port, err := parseServerAddr(s.Config)
		if err != nil {
			return err
		}

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%v", addr, port))
		if err != nil {
			return err
		}

		go s.listen(listener, sshConf)
		return nil
	}

	s.shutdown = make(chan bool)
	s.logger = log.NewLogger(s.Config.Name)
	s.state.Set(service.Starting, "")

	if err := start(); err != nil {
		s.state.Set(service.Error, err.Error())
		return err
	}

	s.state.Set(service.Running, "")
	return nil
}

// Stop stops the SFTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.shutdown <- true
	s.state.Set(service.Offline, "")
	return nil
}

// State returns the state of the SFTP server.
func (s *Server) State() *service.State {
	return &s.state
}
