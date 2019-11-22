package sftp

import (
	"encoding/json"
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

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
	conf := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			user := &model.LocalAccount{Login: c.User()}
			if err := db.Get(user); err != nil {
				return nil, err
			}
			if err := bcrypt.CompareHashAndPassword(user.Password, pass); err != nil {
				return nil, fmt.Errorf("authentication failed (%s)", err)
			}

			return &ssh.Permissions{}, nil
		},
	}

	privateKey, err := ssh.ParsePrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, err
	}
	conf.AddHostKey(privateKey)

	return conf, nil
}

func parseServerAddr(server *model.LocalAgent) (string, uint16, error) {
	conf := &config.SftpProtoConfig{}

	if err := json.Unmarshal(server.ProtoConfig, conf); err != nil {
		return "", 0, err
	}

	return conf.Address, conf.Port, nil
}

func (s *sshServer) toHistory(prog progress) {
	trans := &model.Transfer{ID: prog.ID}
	if err := s.db.Get(trans); err != nil {
		s.logger.Errorf("Error retrieving transfer entry: %s", err)
		return
	}

	ses, err := s.db.BeginTransaction()
	if err != nil {
		s.logger.Errorf("Error starting transaction: %s", err)
		return
	}
	if err := ses.Delete(trans); err != nil {
		s.logger.Errorf("Error deleting the old transfer: %s", err)
		ses.Rollback()
		return
	}

	trans.Status = model.TransferStatus(prog.Status)
	hist, err := trans.ToHistory(s.db, time.Now().UTC())
	if err != nil {
		s.logger.Errorf("Error converting transfer to history: %s", err)
		return
	}

	if err := ses.Create(hist); err != nil {
		s.logger.Errorf("Error inserting new history entry: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Commit(); err != nil {
		s.logger.Errorf("Error committing the transaction: %s", err)
		return
	}
}

func acceptRequests(in <-chan *ssh.Request) {
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
}

func (s *sshServer) logProgress(server *sftp.RequestServer, report <-chan progress,
	done chan bool) {

	go func() {
		select {
		case <-s.shutdown:
			_ = server.Close()
		case <-done:
		}
	}()

	for prog := range report {
		if prog.Status == string(model.StatusTransfer) {
			//trans := &model.Transfer{Progress: prog.Bytes}
			//if err := s.db.Update(trans, prog.ID, false); err != nil {
			//s.logger.Errorf("Error updating transfer: %s", err)
		} else {
			s.toHistory(prog)
		}
	}

	close(done)
}

func (s *sshServer) handleSession(user string, newChannel ssh.NewChannel) {
	if newChannel.ChannelType() != "session" {
		s.logger.Warning("Unknown channel type received")
		_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.logger.Errorf("Failed to accept SFTP session: %s", err)
		return
	}
	go acceptRequests(requests)

	// Resolve conn.User() to model.LocalAccount
	acc := &model.LocalAccount{
		LocalAgentID: s.agent.ID,
		Login:        user,
	}
	if err := s.db.Get(acc); err != nil {
		s.logger.Errorf("Failed to retrieve user: %s", err)
		return
	}

	report := make(chan progress)
	server := sftp.NewRequestServer(channel, makeHandlers(s.db, s.agent,
		acc, report))

	done := make(chan bool)
	go s.logProgress(server, report, done)
	_ = server.Serve()

	close(report)
	<-done
	_ = channel.Close()
}
