package ftp

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/lib/log"
	ftplib "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

var errNotImplemented = errors.New("command not implemented")

type serverFS struct {
	db     *database.DB
	logger *log.Logger
	tracer func() pipeline.Trace

	dbServer *model.LocalAgent
	dbAcc    *model.LocalAccount
}

func (s *serverFS) getFS(path *types.URL) (fs.FS, error) {
	filesys, fsErr := fs.GetFileSystem(s.db, path)
	if fsErr != nil {
		s.logger.Error("Failed to instantiate the filesystem: %v", fsErr)

		//nolint:goerr113 //we mask the internal error
		return nil, errors.New("failed to instantiate the filesystem")
	}

	return filesys, nil
}

func (s *serverFS) Name() string { return s.dbServer.Name }

func (s *serverFS) Mkdir(name string, _ fs.FileMode) error {
	s.logger.Debug(`Received "Mkdir" request on %q`, name)

	return s.mkdirAll(name)
}

func (s *serverFS) MkdirAll(path string, _ fs.FileMode) error {
	s.logger.Debug(`Received "MkdirAll" request on %q`, path)

	return s.mkdirAll(path)
}

//nolint:goerr113 //dynamic errors are used to mask the internal errors (for security reasons)
func (s *serverFS) mkdirAll(path string) error {
	realDir, dirErr := protoutils.GetRealPath(false, s.db, s.logger, s.dbServer, s.dbAcc, path)
	if dirErr != nil {
		s.logger.Error("Failed to build the dir path: %v", dirErr)

		return errors.New("failed to build the dir path")
	}

	filesys, fsErr := s.getFS(realDir)
	if fsErr != nil {
		return fsErr
	}

	if err := fs.MkdirAll(filesys, realDir); err != nil {
		s.logger.Error("Failed to create directory: %v", err)

		return errors.New("failed to create directory") //nolint:goerr113 //too specific
	}

	return nil
}

func (s *serverFS) OpenFile(name string, flags int, _ fs.FileMode) (afero.File, error) {
	s.logger.Debug(`Received "OpenFile" request on %q with flags %s`, name,
		fs.DescribeFlags(flags))

	return s.getHandle(name, flags, 0)
}

func (s *serverFS) GetHandle(name string, flags int, offset int64) (ftplib.FileTransfer, error) {
	s.logger.Debug(`Received "GetHandle" request on %q with flags %s and offset %d`,
		name, fs.DescribeFlags(flags), offset)

	return s.getHandle(name, flags, offset)
}

func (s *serverFS) getHandle(name string, flags int, offset int64) (afero.File, error) {
	if fs.ContainsFlags(flags, fs.FlagAppend) {
		// The "APPEND" command is not allowed.
		return nil, errNotImplemented
	}

	switch flags & (fs.FlagROnly | fs.FlagWOnly | fs.FlagRW) {
	case fs.FlagROnly:
		return s.newServerTransfer(name, true, offset)
	case fs.FlagWOnly:
		return s.newServerTransfer(name, false, offset)
	default:
		s.logger.Error(`Received "OpenFile" request on %q with invalid flags: %d`, name, flags)

		return nil, fmt.Errorf("invalid file flags: %d", flags) //nolint:goerr113 //too specific
	}
}

func (s *serverFS) Open(name string) (afero.File, error) {
	s.logger.Debug(`Received "Open" request on %q`, name)

	return s.newServerTransfer(name, true, 0)
}

func (s *serverFS) Create(name string) (afero.File, error) {
	s.logger.Debug(`Received "Create" request on %q`, name)

	return s.newServerTransfer(name, false, 0)
}

func (s *serverFS) Stat(name string) (fs.FileInfo, error) {
	s.logger.Debug(`Received "Stat" request on %q`, name)

	name = strings.TrimLeft(name, "/")

	info, err := s.stat(name, false)
	if errors.Is(err, fs.ErrNotExist) {
		return s.stat(name, true)
	} else if err != nil {
		return nil, err
	}

	return info, nil
}

//nolint:goerr113 //dynamic errors are used to mask the internal errors (for security reasons)
func (s *serverFS) stat(name string, temp bool) (fs.FileInfo, error) {
	realFile, dirErr := protoutils.GetRealPath(temp, s.db, s.logger, s.dbServer,
		s.dbAcc, name)
	if dirErr != nil {
		return nil, errors.New("failed to build the file path")
	}

	if realFile == nil {
		return protoutils.FakeDirInfo(name), nil
	}

	filesys, fsErr := s.getFS(realFile)
	if fsErr != nil {
		return nil, fsErr
	}

	info, err := fs.Stat(filesys, realFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fs.ErrNotExist
	} else if err != nil {
		return nil, errors.New("failed to stat the file")
	}

	return info, nil
}

//nolint:goerr113 //dynamic errors are used to mask the internal errors (for security reasons)
func (s *serverFS) ReadDir(name string) ([]fs.FileInfo, error) {
	s.logger.Debug(`Received "ReadDir" request on %q`, name)

	name = strings.TrimLeft(name, "/")

	realDir, pathErr := protoutils.GetRealPath(false, s.db, s.logger, s.dbServer,
		s.dbAcc, name)
	if pathErr != nil {
		if realDir, pathErr = protoutils.GetRealPath(true, s.db, s.logger, s.dbServer,
			s.dbAcc, name); pathErr != nil {
			return nil, fmt.Errorf("failed to build the dir path: %w", pathErr)
		}
	}

	if realDir == nil {
		infos, err := protoutils.GetRulesPaths(s.db, s.dbServer, s.dbAcc, name)
		if err != nil {
			s.logger.Error("Failed to retrieve rules: %v", err)

			return nil, errors.New("failed to retrieve rules")
		}

		for _, entry := range infos {
			s.logger.Debug("Returned dir entry: %q", entry.Name())
		}

		return infos, nil
	}

	filesys, fsErr := s.getFS(realDir)
	if fsErr != nil {
		return nil, fsErr
	}

	entries, dirErr := fs.ReadDir(filesys, realDir)
	if dirErr != nil {
		s.logger.Error(`Failed to list directory "%s": %v`, realDir, dirErr)

		return nil, fmt.Errorf(`failed to list directory %q`, name)
	}

	infos := make([]fs.FileInfo, len(entries))

	for i, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			s.logger.Error("Failed to retrieve the file info: %v", infoErr)

			return nil, errors.New("failed to retrieve the file info")
		}

		s.logger.Debug("Returned dir entry: %q", info.Name())

		infos[i] = info
	}

	return infos, nil
}

// The following methods are not implemented. Some might be implemented later
// if there is a legitimate need for it.

func (*serverFS) Remove(string) error                        { return errNotImplemented }
func (*serverFS) RemoveAll(string) error                     { return errNotImplemented }
func (*serverFS) Rename(string, string) error                { return errNotImplemented }
func (*serverFS) Chmod(string, fs.FileMode) error            { return errNotImplemented }
func (*serverFS) Chown(string, int, int) error               { return errNotImplemented }
func (*serverFS) Chtimes(string, time.Time, time.Time) error { return errNotImplemented }
