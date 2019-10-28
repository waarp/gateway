package sftp

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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
	config := &ssh.ServerConfig{
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

func (s *sshServer) toHistory() {
	for res := range s.results {
		hist, err := res.ToHistory(s.db, time.Now().UTC())
		if err != nil {
			s.logger.Errorf("Error while converting transfer to history: %s", err)
			continue
		}

		if res.error != nil {
			hist.Status = model.StatusError
		} else {
			hist.Status = model.StatusDone
		}

		ses, err := s.db.BeginTransaction()
		if err != nil {
			s.logger.Errorf("Error while starting transaction: %s", err)
			continue
		}
		if err := ses.Create(hist); err != nil {
			s.logger.Errorf("Error while inserting new history entry: %s", err)
			ses.Rollback()
			continue
		}
		if err := ses.Delete(res.Transfer); err != nil {
			s.logger.Errorf("Error while deleting the old transfer: %s", err)
			ses.Rollback()
			continue
		}
		if err := ses.Commit(); err != nil {
			s.logger.Errorf("Error while committing the transaction: %s", err)
			continue
		}
	}
	close(s.finished)
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

func (s *sshServer) handleAnswer(server *sftp.RequestServer, report chan *model.Transfer,
	errReport <-chan error) {

	select {
	case <-s.shutdown:
		_ = server.Close()
		if trans, ok := <-report; ok {
			s.results <- result{Transfer: trans, error: fmt.Errorf("server shutdown")}
		}
	case err := <-errReport:
		trans, ok := <-report
		if ok {
			if err == io.EOF {
				err = nil
			}
			s.results <- result{Transfer: trans, error: err}
		}
		_ = server.Close()
		close(report)
	}
}
