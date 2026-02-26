package ftp

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	ftplib "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errNotImplemented = errors.New("command not implemented")

type serverFS struct {
	db     *database.DB
	logger *log.Logger
	tracer func() pipeline.Trace

	dbServer *model.LocalAgent
	dbAcc    *model.LocalAccount
}

func (s *serverFS) Name() string { return s.dbServer.Name }

func (s *serverFS) Mkdir(name string, _ os.FileMode) error {
	s.logger.Debugf(`Received "Mkdir" request on %q`, name)

	return s.mkdirAll(name)
}

func (s *serverFS) MkdirAll(path string, _ os.FileMode) error {
	s.logger.Debugf(`Received "MkdirAll" request on %q`, path)

	return s.mkdirAll(path)
}

//nolint:err113 //dynamic errors are used to mask the internal errors (for security reasons)
func (s *serverFS) mkdirAll(path string) error {
	realDir, dirErr := protoutils.GetRealPath(false, s.db, s.logger, s.dbServer, s.dbAcc, path)
	if dirErr != nil {
		s.logger.Errorf("Failed to build the dir path: %v", dirErr)

		return errors.New("failed to build the dir path")
	}

	if err := fs.MkdirAll(realDir); err != nil {
		s.logger.Errorf("Failed to create directory: %v", err)

		return errors.New("failed to create directory") //nolint:err113 //too specific
	}

	return nil
}

func (s *serverFS) OpenFile(name string, flags int, _ os.FileMode) (afero.File, error) {
	s.logger.Debugf(`Received "OpenFile" request on %q with flags %s`, name,
		utils.DescribeFlags(flags))

	return s.getHandle(name, flags, 0)
}

func (s *serverFS) GetHandle(name string, flags int, offset int64) (ftplib.FileTransfer, error) {
	s.logger.Debugf(`Received "GetHandle" request on %q with flags %s and offset %d`,
		name, utils.DescribeFlags(flags), offset)

	return s.getHandle(name, flags, offset)
}

func (s *serverFS) getHandle(name string, flags int, offset int64) (afero.File, error) {
	switch flags & (os.O_RDONLY | os.O_WRONLY | os.O_RDWR) {
	case os.O_RDONLY:
		return s.newServerTransfer(name, true, offset)
	case os.O_WRONLY:
		return s.newServerTransfer(name, false, offset)
	default:
		s.logger.Errorf(`Received "OpenFile" request on %q with invalid flags: %d`, name, flags)

		return nil, fmt.Errorf("invalid file flags: %d", flags) //nolint:err113 //too specific
	}
}

func (s *serverFS) Open(name string) (afero.File, error) {
	s.logger.Debugf(`Received "Open" request on %q`, name)

	return s.newServerTransfer(name, true, 0)
}

func (s *serverFS) Create(name string) (afero.File, error) {
	s.logger.Debugf(`Received "Create" request on %q`, name)

	return s.newServerTransfer(name, false, 0)
}

func (s *serverFS) Stat(name string) (os.FileInfo, error) {
	s.logger.Debugf(`Received "Stat" request on %q`, name)

	name = strings.TrimLeft(name, "/")

	info, err := s.stat(name, false)
	if errors.Is(err, fs.ErrNotExist) {
		return s.stat(name, true)
	} else if err != nil {
		return nil, err
	}

	return info, nil
}

//nolint:err113 //dynamic errors are used to mask the internal errors (for security reasons)
func (s *serverFS) stat(name string, temp bool) (os.FileInfo, error) {
	realFile, dirErr := protoutils.GetRealPath(temp, s.db, s.logger, s.dbServer,
		s.dbAcc, name)
	if dirErr != nil {
		return nil, errors.New("failed to build the file path")
	}

	if realFile == "" {
		return protoutils.FakeDirInfo(name), nil
	}

	info, err := fs.Stat(realFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fs.ErrNotExist
	} else if err != nil {
		return nil, errors.New("failed to stat the file")
	}

	return info, nil
}

//nolint:err113 //dynamic errors are used to mask the internal errors (for security reasons)
func (s *serverFS) ReadDir(name string) ([]os.FileInfo, error) {
	s.logger.Debugf(`Received "ReadDir" request on %q`, name)

	name = strings.TrimLeft(name, "/")

	realDir, pathErr := protoutils.GetRealPath(false, s.db, s.logger, s.dbServer,
		s.dbAcc, name)
	if pathErr != nil {
		if realDir, pathErr = protoutils.GetRealPath(true, s.db, s.logger, s.dbServer,
			s.dbAcc, name); pathErr != nil {
			return nil, fmt.Errorf("failed to build the dir path: %w", pathErr)
		}
	}

	if realDir == "" {
		infos, err := protoutils.GetRulesPaths(s.db, s.dbServer, s.dbAcc, name)
		if err != nil {
			s.logger.Errorf("Failed to retrieve rules: %v", err)

			return nil, errors.New("failed to retrieve rules")
		}

		for _, entry := range infos {
			s.logger.Debugf("Returned dir entry: %q", entry.Name())
		}

		return infos.AsFileInfos(), nil
	}

	entries, dirErr := fs.List(realDir)
	if dirErr != nil {
		s.logger.Errorf(`Failed to list directory "%s": %v`, realDir, dirErr)

		return nil, fmt.Errorf(`failed to list directory %q`, name)
	}

	infos := make([]os.FileInfo, len(entries))

	for i, entry := range entries {
		s.logger.Debugf("Returned dir entry: %q", entry.Name())

		var err error
		if infos[i], err = entry.Info(); err != nil {
			s.logger.Errorf("Failed to retrieve the file %q info: %v", entry.Name(), err)

			return infos, fmt.Errorf("failed to retrieve file %q info", entry.Name())
		}
	}

	return infos, nil
}

// The following methods are not implemented. Some might be implemented later
// if there is a legitimate need for it.

func (*serverFS) Remove(string) error                    { return errNotImplemented }
func (*serverFS) RemoveAll(string) error                 { return errNotImplemented }
func (*serverFS) Rename(_, _ string) error               { return errNotImplemented }
func (*serverFS) Chmod(string, os.FileMode) error        { return errNotImplemented }
func (*serverFS) Chown(_ string, _, _ int) error         { return errNotImplemented }
func (*serverFS) Chtimes(_ string, _, _ time.Time) error { return errNotImplemented }
