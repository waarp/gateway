package as2

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/dustin/go-humanize"
	"github.com/pbnjay/memory"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const defaultBufSize = 1000000 // 1MB

var (
	//nolint:gochecknoglobals //we need to keep track of the total memory
	totalMemory = memory.TotalMemory()

	ErrMaxSizeTooBig = errors.New("max file size cannot exceed the available system memory")
)

type BufferSize int64

//nolint:wrapcheck //no need to wrap here
func (s *BufferSize) UnmarshalJSON(b []byte) error {
	var size uint64
	if err := json.Unmarshal(b, &size); err != nil {
		return err
	}

	if size > totalMemory {
		return ErrMaxSizeTooBig
	}

	*s = BufferSize(size)

	return nil
}

func getFileContent(file io.Reader, bufSize uint64) ([]byte, *pipeline.Error) {
	r := io.LimitReader(file, int64(bufSize)+1)

	cont, err := io.ReadAll(r)
	if err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal, "failed to read file content")
	}

	if uint64(len(cont)) > bufSize {
		return nil, pipeline.NewErrorf(types.TeExceededLimit,
			"file size exceeds the maximum allowed size of %s",
			humanize.Bytes(bufSize))
	}

	return cont, nil
}
