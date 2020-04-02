package sftp

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/pkg/sftp"
)

type sftpStream struct {
	*pipeline.TransferStream

	transErr model.TransferError
}

// modelToSFTP converts the given TransferError into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most TransferErrors will be converted to the generic code SSH_FX_FAILURE.
func modelToSFTP(err model.TransferError) error {
	switch err.Code {
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
	default:
		return sftp.ErrSSHFxFailure
	}
}

// newSftpStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func newSftpStream(logger *log.Logger, db *database.Db, root string,
	trans model.Transfer) (*sftpStream, error) {

	s, err := pipeline.NewTransferStream(logger, db, root, trans)
	if err.Code != model.TeOk {
		return nil, modelToSFTP(err)
	}
	stream := &sftpStream{TransferStream: s}

	if te := s.Start(); te.Code != model.TeOk {
		s.ErrorTasks(te)
		s.Exit()
		return nil, modelToSFTP(te)
	}

	if pe := s.PreTasks(); pe.Code != model.TeOk {
		s.ErrorTasks(pe)
		s.Exit()
		return nil, modelToSFTP(pe)
	}

	return stream, nil
}

func (s *sftpStream) TransferError(err error) {
	if te, ok := err.(model.TransferError); ok {
		s.transErr = te
		return
	}
	s.transErr = model.NewTransferError(model.TeConnectionReset,
		"SFTP connection closed unexpectedly")
}

func (s *sftpStream) Close() error {
	defer s.Exit()

	if err := s.TransferStream.Close(); err != nil {
		s.Logger.Warningf("Failed to close local file: %s", err.Error())
	}

	if s.transErr.Code == model.TeOk {
		s.transErr = s.PostTasks()
		if s.transErr.Code == model.TeOk {
			return nil
		}
	}

	s.ErrorTasks(s.transErr)
	return modelToSFTP(s.transErr)
}
