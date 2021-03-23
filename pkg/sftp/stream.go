package sftp

import (
	"fmt"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

type errorHandler struct {
	ch ssh.Channel
}

func (e *errorHandler) SendError(error) {
	_ = e.ch.CloseWrite()
}

type stream struct {
	pipeline *pipeline.Pipeline
	file     pipeline.TransferStream
}

// newStream initialises a special kind of TransferStream tailored for
// the SFTP server. This constructor initialises a TransferStream, opens the
// local file and executes the pre-tasks.
func newStream(db *database.DB, paths *conf.PathsConfig, trans *model.Transfer,
	handler pipeline.Server) (*stream, error) {

	pip, err := pipeline.NewServerPipeline(db, paths, trans, handler)
	if err != nil {
		return nil, modelToSFTP(err)
	}

	str := &stream{pipeline: pip}

	if err := pip.PreTasks(); err != nil {
		return str, modelToSFTP(err)
	}

	file, err := pip.StartData()
	if err != nil {
		return str, modelToSFTP(err)
	}
	str.file = file

	return str, nil
}

func (s *stream) TransferError(err error) {
	if err == io.EOF {
		s.pipeline.SetError(fmt.Errorf("session closed by remote host"))
	} else {
		s.pipeline.SetError(err)
	}
}

func (s *stream) ReadAt(p []byte, off int64) (int, error) {
	n, err := s.file.ReadAt(p, off)
	if err != nil && err != io.EOF {
		return n, modelToSFTP(err)
	}
	return n, err
}

func (s *stream) WriteAt(p []byte, off int64) (int, error) {
	n, err := s.file.WriteAt(p, off)
	if err != nil {
		return n, modelToSFTP(err)
	}
	return n, nil
}

func (s *stream) close() error {
	if err := s.pipeline.EndData(); err != nil {
		return err
	}
	if err := s.pipeline.PostTasks(); err != nil {
		return err
	}
	if err := s.pipeline.EndTransfer(); err != nil {
		return err
	}
	return nil
}

func (s *stream) Close() error {
	if err := s.close(); err != nil {
		return modelToSFTP(err)
	}
	return nil
}
