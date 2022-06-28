package r66

import (
	"errors"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
)

type clientStream struct {
	stream pipeline.DataStream
}

func (c *clientStream) ReadAt(p []byte, off int64) (int, error) {
	n, err := c.stream.ReadAt(p, off)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		return n, internal.ToR66Error(err)
	}

	return n, nil
}

func (c *clientStream) WriteAt(p []byte, off int64) (int, error) {
	n, err := c.stream.WriteAt(p, off)
	if err != nil {
		return n, internal.ToR66Error(err)
	}

	return n, nil
}
