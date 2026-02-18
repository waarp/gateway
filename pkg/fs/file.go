package fs

import (
	"io"
)

func ReadFullFile(path string) ([]byte, error) {
	file, openErr := Open(path)
	if openErr != nil {
		return nil, openErr
	}
	//nolint:errcheck //closing a read only file never returns an error
	defer file.Close()

	cont, rdErr := io.ReadAll(file)
	if rdErr != nil {
		return nil, pathError("read", path, rdErr)
	}

	return cont, nil
}

func WriteFullFile(path string, content []byte) error {
	file, openErr := Create(path)
	if openErr != nil {
		return openErr
	}

	if _, wrErr := file.Write(content); wrErr != nil {
		return pathError("write", path, wrErr)
	}

	if err := file.Close(); err != nil {
		return pathError("close", path, err)
	}

	return nil
}

func WriteFileFromReader(path string, reader io.Reader) error {
	file, openErr := Create(path)
	if openErr != nil {
		return openErr
	}

	if _, wrErr := io.Copy(file, reader); wrErr != nil {
		return pathError("write", path, wrErr)
	}

	if err := file.Close(); err != nil {
		return pathError("close", path, err)
	}

	return nil
}
