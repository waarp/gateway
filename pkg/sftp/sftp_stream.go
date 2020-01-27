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

	transErr *model.PipelineError
}

// modelToSFTP converts the given PipelineError into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func modelToSFTP(err *model.PipelineError) error {
	if err.Kind == model.KindTransfer {
		switch err.Cause.Code {
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
	return sftp.ErrSSHFxFailure
}

// newSftpStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func newSftpStream(logger *log.Logger, db *database.Db, root string,
	trans model.Transfer) (*sftpStream, error) {

	s, err := pipeline.NewTransferStream(logger, db, root, trans)
	if err != nil {
		return nil, sftp.ErrSSHFxFailure
	}
	stream := &sftpStream{TransferStream: s}

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
	if te, ok := err.(*model.PipelineError); ok {
		s.transErr = te
		return
	}
	s.transErr = model.NewPipelineError(model.TeConnectionReset,
		"SFTP connection closed unexpectedly")
}

func (s *sftpStream) Close() error {
	if e := s.TransferStream.Close(); e != nil {
		s.Logger.Warningf("Failed to close the local file: %s", e.Error())
	}

	if s.transErr == nil {
		s.transErr = s.PostTasks()
		if s.transErr == nil {
			s.Transfer.Status = model.StatusDone
			s.Archive()
			s.Exit()
			return nil
		}
	}

	pipeline.HandleError(s.TransferStream, s.transErr)
	return modelToSFTP(s.transErr)
}
