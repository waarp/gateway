package sftp

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func makeDummyServer(pk, pbk, login, password string) (net.Listener, error) {
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pbk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse user public key: %s", err)
	}

	conf := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() == login && bytes.Equal(key.Marshal(), publicKey.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("public key '%s' rejected for user '%s'", key.Type(), conn.User())
		},
		PasswordCallback: func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if conn.User() == login && string(pass) == password {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("password '%s' rejected for user '%s'", pass, conn.User())
		},
	}

	privateKey, err := ssh.ParsePrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SFTP server key: %s", err)
	}

	conf.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start SFTP server: %s", err)
	}

	go handleSFTP(listener, conf)

	return listener, nil
}

func handleSFTP(listener net.Listener, config *ssh.ServerConfig) {
	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept incoming connection", err)
			continue
		}

		_, chans, reqs, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			log.Println("Failed to handshake", err)
			continue
		}

		go ssh.DiscardRequests(reqs)

		for newChannel := range chans {
			if newChannel.ChannelType() != "session" {
				_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}
			channel, requests, err := newChannel.Accept()
			if err != nil {
				log.Println("Could not accept channel.", err)
				break
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

			server, err := sftp.NewServer(channel)
			if err != nil {
				log.Println("Failed to start SFTP server", err)
				break
			}
			if err := server.Serve(); err == io.EOF {
				_ = server.Close()
				break
			} else if err != nil {
				log.Println("sftp server completed with error:", err)
				break
			}
		}
	}
}
