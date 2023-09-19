package utils

import "io"

func AsReaderAt(r io.Reader) io.ReaderAt { return &readerAt{r} }
func AsWriterAt(r io.Writer) io.WriterAt { return &writerAt{r} }

type readerAt struct{ io.Reader }

//nolint:wrapcheck //this is just a wrapper around io.Reader, errors should not be altered
func (r *readerAt) ReadAt(p []byte, _ int64) (int, error) {
	return r.Read(p)
}

type writerAt struct{ io.Writer }

//nolint:wrapcheck //this is just a wrapper around io.Writer, errors should not be altered
func (w *writerAt) WriteAt(p []byte, _ int64) (int, error) {
	return w.Write(p)
}
