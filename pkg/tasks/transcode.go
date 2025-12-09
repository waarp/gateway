package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"golang.org/x/text/transform"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/ordered"
)

//nolint:gochecknoglobals //global var is needed here for future-proofing
var TranscodeFormats = ordered.Map[string, encoding.Encoding]{}

//nolint:gochecknoinits //init is needed here to populate TranscodeFormats
func init() {
	TranscodeFormats.Add("UTF-8", unicode.UTF8)
	TranscodeFormats.Add("UTF-8 BOM", unicode.UTF8BOM)
	TranscodeFormats.Add("UTF-16BE", unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM))
	TranscodeFormats.Add("UTF-16LE", unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM))
	TranscodeFormats.Add("UTF-16BE BOM", unicode.UTF16(unicode.BigEndian, unicode.UseBOM))
	TranscodeFormats.Add("UTF-16LE BOM", unicode.UTF16(unicode.LittleEndian, unicode.UseBOM))
	TranscodeFormats.Add("UTF-32BE", utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM))
	TranscodeFormats.Add("UTF-32LE", utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM))
	TranscodeFormats.Add("UTF-32BE BOM", utf32.UTF32(utf32.BigEndian, utf32.UseBOM))
	TranscodeFormats.Add("UTF-32LE BOM", utf32.UTF32(utf32.LittleEndian, utf32.UseBOM))
	TranscodeFormats.Add("IBM Code Page 273", newEBCDICEncoding(ebcdic273))
	TranscodeFormats.Add("IBM Code Page 500", newEBCDICEncoding(ebcdic500))
	TranscodeFormats.Add("IBM Code Page 1141", newEBCDICEncoding(ebcdic1141))
	TranscodeFormats.Add("IBM Code Page 1148", newEBCDICEncoding(ebcdic1148))

	for _, chMap := range charmap.All {
		//nolint:errcheck,forcetypeassert //this assertion always succeeds
		name := chMap.(fmt.Stringer).String()
		TranscodeFormats.Add(name, chMap)
	}
}

var (
	ErrTranscodeNoSrcEncoding      = errors.New("missing source encoding")
	ErrTranscodeNoDstEncoding      = errors.New("missing destination encoding")
	ErrTranscodeInvalidEncoding    = errors.New("invalid encoding")
	ErrTranscodeIdenticalEncodings = errors.New("source and destination encodings are identical")
)

type transcodeTask struct {
	FromCharset string `json:"fromCharset"`
	ToCharset   string `json:"toCharset"`

	from encoding.Encoding
	to   encoding.Encoding
}

func getEncoding(charset string) (encoding.Encoding, error) {
	if encoder, ok := TranscodeFormats.Get(charset); ok {
		return encoder, nil
	}

	return nil, fmt.Errorf("%w %q", ErrTranscodeInvalidEncoding, charset)
}

func (t *transcodeTask) parseParams(params map[string]string) error {
	if utils.JSONConvert(params, t) != nil {
		return fmt.Errorf("failed to parse transcode task params: %w", ErrBadTaskArguments)
	}

	if t.FromCharset == "" {
		return ErrTranscodeNoSrcEncoding
	}

	if t.ToCharset == "" {
		return ErrTranscodeNoDstEncoding
	}

	var err error

	if t.from, err = getEncoding(t.FromCharset); err != nil {
		return fmt.Errorf("source encoding: %w", err)
	}

	if t.to, err = getEncoding(t.ToCharset); err != nil {
		return fmt.Errorf("destination encoding: %w", err)
	}

	if t.ToCharset == t.FromCharset {
		return ErrTranscodeIdenticalEncodings
	}

	return nil
}

func (t *transcodeTask) Validate(params map[string]string) error {
	return t.parseParams(params)
}

func (t *transcodeTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.parseParams(params); err != nil {
		return err
	}

	tempFilename := transCtx.Transfer.LocalPath + ".tmp"
	if err := t.transcode(logger, transCtx.Transfer.LocalPath, tempFilename); err != nil {
		return err
	}

	if err := fs.Remove(transCtx.Transfer.LocalPath); err != nil {
		logger.Errorf("Failed to delete source file: %v", err)

		return fmt.Errorf("failed to delete source file: %w", err)
	}

	if err := fs.MoveFile(tempFilename, transCtx.Transfer.LocalPath); err != nil {
		logger.Errorf("Failed to rename temporary file: %v", err)

		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

func (t *transcodeTask) transcode(logger *log.Logger, srcFilepath, dstFilepath string) error {
	srcFile, opErr := fs.Open(srcFilepath)
	if opErr != nil {
		logger.Errorf("Failed to open source file: %v", opErr)

		return fmt.Errorf("failed to open source file: %w", opErr)
	}
	defer srcFile.Close() //nolint:errcheck //close never returns an error on read-only files

	dstFile, opErr := fs.Create(dstFilepath)
	if opErr != nil {
		logger.Errorf("Failed to create destination file: %v", opErr)

		return fmt.Errorf("failed to create destination file: %w", opErr)
	}
	defer dstFile.Close() //nolint:errcheck //error is checked bellow, this is just in case of error

	src := transform.NewReader(srcFile, t.from.NewDecoder())
	dst := transform.NewWriter(dstFile, t.to.NewEncoder())

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to transcode file: %w", err)
	}

	if err := dst.Close(); err != nil {
		logger.Errorf("Failed to close destination file: %v", err)

		return fmt.Errorf("failed to close destination file: %w", err)
	}

	return nil
}
