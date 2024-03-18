package fs

import (
	"io"
	"io/fs"
)

type Seeker interface {
	CurrentOffset() int64
	Stat() (FileInfo, error)
}

func GetSeekNewOffset(seeker Seeker, offset int64, whence int) (int64, error) {
	newOffset := seeker.CurrentOffset()

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset += offset
	case io.SeekEnd:
		info, statErr := seeker.Stat()
		if statErr != nil {
			//nolint:wrapcheck //wrapping the error here adds nothing
			return 0, statErr
		}

		newOffset = info.Size() + offset
	default:
		return 0, fs.ErrInvalid
	}

	if newOffset < 0 {
		return 0, fs.ErrInvalid
	}

	return newOffset, nil
}
