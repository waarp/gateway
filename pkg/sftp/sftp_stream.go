package sftp

import (
	"context"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type sftpStream struct {
	*pipeline.TransferStream

	transErr error
}

// modelToSFTP converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func modelToSFTP(err error) error {
	if tErr, ok := err.(types.TransferError); ok {
		switch tErr.Code {
		case types.TeOk:
			return sftp.ErrSSHFxOk
		case types.TeUnimplemented:
			return sftp.ErrSSHFxOpUnsupported
		case types.TeIntegrity:
			return sftp.ErrSSHFxBadMessage
		case types.TeFileNotFound:
			return sftp.ErrSSHFxNoSuchFile
		case types.TeForbidden:
			return sftp.ErrSSHFxPermissionDenied
		}
	}
	return err
}

// newSftpStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func newSftpStream(ctx context.Context, logger *log.Logger, db *database.DB,
	paths pipeline.Paths, trans model.Transfer) (*sftpStream, error) {

	s, err := pipeline.NewTransferStream(ctx, logger, db, paths, &trans)
	if err != nil {
		return nil, modelToSFTP(err)
	}
	stream := &sftpStream{TransferStream: s}

	s.Logger.Infof("Beginning transfer n°%d", trans.ID)
	if pe := s.PreTasks(); pe != nil {
		pipeline.HandleError(s, pe)
		return nil, modelToSFTP(pe)
	}

	return stream, nil
}

func (s *sftpStream) TransferError(error) {
	select {
	case <-s.Ctx.Done():
		s.transErr = &model.ShutdownError{}
		return
	default:
	}
	if s.transErr == nil {
		switch s.Transfer.Step {
		case types.StepPreTasks:
			s.transErr = types.NewTransferError(types.TeExternalOperation,
				"Remote pre-tasks failed")
		case types.StepData:
			s.transErr = types.NewTransferError(types.TeConnectionReset,
				"SFTP connection closed unexpectedly")
		case types.StepPostTasks:
			s.transErr = types.NewTransferError(types.TeExternalOperation,
				"Remote post-tasks failed")
		}
	}
}

func (s *sftpStream) ReadAt(p []byte, off int64) (int, error) {
	if s.transErr != nil {
		return 0, modelToSFTP(s.transErr)
	}
	if s.File == nil {
		if te := s.Start(); te != nil {
			pipeline.HandleError(s.TransferStream, te)
			return 0, modelToSFTP(te)
		}
	}

	n, err := s.TransferStream.ReadAt(p, off)
	if err != nil && err != io.EOF {
		s.transErr = err
		_ = s.Close()
		return n, modelToSFTP(s.transErr)
	}
	return n, err
}

func (s *sftpStream) WriteAt(p []byte, off int64) (int, error) {
	if s.transErr != nil {
		return 0, modelToSFTP(s.transErr)
	}
	if s.File == nil {
		if te := s.Start(); te != nil {
			pipeline.HandleError(s.TransferStream, te)
			return 0, modelToSFTP(te)
		}
	}

	n, err := s.TransferStream.WriteAt(p, off)
	if err != nil {
		s.transErr = err
		_ = s.Close()
		return n, modelToSFTP(s.transErr)
	}
	return n, nil
}

func (s *sftpStream) Close() error {
	if s.TransferStream.File != nil {
		if err := s.TransferStream.Close(); err != nil {
			s.transErr = err
		}
	}

	if s.Transfer.Step >= types.StepData && s.transErr == nil {
		if err := s.TransferStream.Move(); err != nil {
			s.transErr = err
		}
	}

	if s.transErr == nil {
		s.transErr = s.PostTasks()
		if s.transErr == nil {
			s.Transfer.Step = types.StepNone
			s.Transfer.Status = types.StatusDone
			s.Transfer.TaskNumber = 0
			if s.Archive() == nil {
				s.Logger.Info("Execution finished without errors")
			}
			return nil
		}
	}

	pipeline.HandleError(s.TransferStream, s.transErr)
	return modelToSFTP(s.transErr)
}
