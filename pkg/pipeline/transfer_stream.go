package pipeline

import (
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func Leaf(s string) utils.Leaf     { return utils.Leaf(s) }
func Branch(s string) utils.Branch { return utils.Branch(s) }

type TransferStream interface {
	io.Reader
	io.Writer
	io.ReaderAt
	io.WriterAt
	close() error
	move() error
	stop()
}

type nullStream struct{}

func (*nullStream) Read([]byte) (int, error)               { return 0, io.EOF }
func (*nullStream) Write(p []byte) (int, error)            { return len(p), nil }
func (*nullStream) ReadAt([]byte, int64) (int, error)      { return 0, io.EOF }
func (*nullStream) WriteAt(p []byte, _ int64) (int, error) { return len(p), nil }
func (*nullStream) close() error                           { return nil }
func (*nullStream) move() error                            { return nil }
func (*nullStream) stop()                                  {}
