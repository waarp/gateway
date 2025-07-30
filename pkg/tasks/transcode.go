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
)

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
	switch charset {
	case "UTF-8":
		return unicode.UTF8, nil
	case "UTF-8 BOM":
		return unicode.UTF8BOM, nil
	case "UTF-16BE":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	case "UTF-16LE":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case "UTF-16BE BOM":
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM), nil
	case "UTF-16LE BOM":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), nil
	case "UTF-32BE":
		return utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM), nil
	case "UTF-32LE":
		return utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM), nil
	case "UTF-32BE BOM":
		return utf32.UTF32(utf32.BigEndian, utf32.UseBOM), nil
	case "UTF-32LE BOM":
		return utf32.UTF32(utf32.LittleEndian, utf32.UseBOM), nil
	case "IBM Code Page 273":
		return newEBCDICEncoding(ebcdic273), nil
	case "IBM Code Page 500":
		return newEBCDICEncoding(ebcdic500), nil
	case "IBM Code Page 1141":
		return newEBCDICEncoding(ebcdic1141), nil
	case "IBM Code Page 1148":
		return newEBCDICEncoding(ebcdic1148), nil
	default:
		for _, chMap := range charmap.All {
			//nolint:errcheck,forcetypeassert //this assertion always succeeds
			if name := chMap.(fmt.Stringer); name.String() == charset {
				return chMap, nil
			}
		}

		return nil, fmt.Errorf("%w %q", ErrTranscodeInvalidEncoding, charset)
	}
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

func (t *transcodeTask) Run(_ context.Context, params map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
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
