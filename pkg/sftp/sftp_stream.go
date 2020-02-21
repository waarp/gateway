package sftp

import (
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sftpStream struct {
	*pipeline.TransferStream

	transErr *model.PipelineError
	servConn *ssh.ServerConn
}

// modelToSFTP converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func modelToSFTP(err error) error {
	if pErr, ok := err.(*model.PipelineError); ok {
		if pErr.Kind == model.KindTransfer {
			switch pErr.Cause.Code {
			case model.TeOk:
				return sftp.ErrSSHFxOk
			case model.TeUnimplemented:
				return sftp.ErrSSHFxOpUnsupported
			case model.TeIntegrity:
				return sftp.ErrSSHFxBadMessage
			case model.TeFileNotFound:
				return sftp.ErrSSHFxNoSuchFile
			case model.TeForbidden:
				return sftp.ErrSSHFxPermissionDenied
			}
		}
	}
	return err
}

// newSftpStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func newSftpStream(logger *log.Logger, db *database.Db, root string,
	trans model.Transfer, servConn *ssh.ServerConn) (*sftpStream, error) {

	s, err := pipeline.NewTransferStream(logger, db, root, trans)
	if err != nil {
		return nil, modelToSFTP(err)
	}
	stream := &sftpStream{TransferStream: s, servConn: servConn}

	if te := s.Start(); te != nil {
		pipeline.HandleError(s, te)
		return nil, modelToSFTP(te)
	}

	if pe := s.PreTasks(); pe != nil {
		pipeline.HandleError(s, pe)
		return nil, modelToSFTP(pe)
	}

	return stream, nil
}

func (s *sftpStream) TransferError(err error) {
	if s.transErr == nil {
		switch s.Transfer.Step {
		case model.StepPreTasks:
			s.transErr = model.NewPipelineError(model.TeExternalOperation,
				"Remote pre-tasks failed")
		case model.StepData:
			s.transErr = model.NewPipelineError(model.TeConnectionReset,
				"SFTP connection closed unexpectedly")
		case model.StepPostTasks:
			s.transErr = model.NewPipelineError(model.TeExternalOperation,
				"Remote post-tasks failed")
		}
	}
}

func (s *sftpStream) ReadAt(p []byte, off int64) (int, error) {
	if s.transErr != nil {
		return 0, modelToSFTP(s.transErr)
	}
	n, err := s.TransferStream.ReadAt(p, off)
	if err == io.EOF {
		s.Transfer.Progress = 0
		s.Transfer.Step = model.StepPostTasks
		if dbErr := s.Transfer.Update(s.Db); dbErr != nil {
			return 0, dbErr
		}
		return n, err
	}
	if err != nil {
		pErr := err.(*model.PipelineError)
		s.transErr = pErr
		if pErr.Kind != model.KindTransfer {
			_ = s.Close()
		}
		return n, modelToSFTP(s.transErr)
	}
	return n, err
}

func (s *sftpStream) WriteAt(p []byte, off int64) (int, error) {
	if s.transErr != nil {
		return 0, modelToSFTP(s.transErr)
	}
	if len(p) == 0 {
		s.Transfer.Progress = 0
		s.Transfer.Step = model.StepPostTasks
		if err := s.Transfer.Update(s.Db); err != nil {
			return 0, err
		}
	}

	n, err := s.TransferStream.WriteAt(p, off)
	if err != nil {
		pErr := err.(*model.PipelineError)
		s.transErr = pErr
		if pErr.Kind != model.KindTransfer {
			_ = s.Close()
		}
		return n, modelToSFTP(s.transErr)
	}
	return n, nil
}

func (s *sftpStream) Close() error {
	if s.transErr == nil {
		s.transErr = s.PostTasks()
		if s.transErr == nil {
			s.transErr = s.Finalize()
			if s.transErr == nil {
				s.Transfer.Status = model.StatusDone
				s.Archive()
				s.Exit()
				return nil
			}
		}
	}

	pipeline.HandleError(s.TransferStream, s.transErr)
	if s.transErr.Kind == model.KindInterrupt {
		_ = s.servConn.Close()
	}
	return modelToSFTP(s.transErr)
}
