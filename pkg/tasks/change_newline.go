package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrChNewlineMissingFrom = errors.New(`missing "from" argument`)
	ErrChNewlineMissingTo   = errors.New(`missing "to" argument`)
)

type chNewlineTask struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (c *chNewlineTask) parseParams(args map[string]string) error {
	*c = chNewlineTask{}
	if err := utils.JSONConvert(args, c); err != nil {
		return fmt.Errorf("failed to parse the change newline parameters: %w", err)
	}

	if c.From == "" {
		return ErrChNewlineMissingFrom
	}

	if c.To == "" {
		return ErrChNewlineMissingTo
	}

	return nil
}

func (c *chNewlineTask) Validate(args map[string]string) error {
	return c.parseParams(args)
}

func (c *chNewlineTask) Run(ctx context.Context, args map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := c.parseParams(args); err != nil {
		return err
	}

	if err := c.run(ctx, transCtx); err != nil {
		logger.Errorf("Failed to change newline: %v", err)

		return err
	}

	logger.Debugf("Changed newline separator from %q to %q", c.From, c.To)

	return nil
}

func (c *chNewlineTask) run(ctx context.Context, transCtx *model.TransferContext) error {
	srcFilepath := transCtx.Transfer.LocalPath
	dstFilepath := transCtx.Transfer.LocalPath + ".newline"

	if err := c.doChangeNewline(ctx, srcFilepath, dstFilepath); err != nil {
		return err
	}

	if err := fs.Remove(srcFilepath); err != nil {
		return fmt.Errorf("failed to remove old file: %w", err)
	}

	if err := fs.MoveFile(dstFilepath, srcFilepath); err != nil {
		return fmt.Errorf("failed to rename new file: %w", err)
	}

	return nil
}

func (c *chNewlineTask) doChangeNewline(ctx context.Context,
	srcFilepath, dstFilepath string,
) error {
	const bufSize = 32 * 1024
	buf := make([]byte, bufSize)

	srcFile, opErr := fs.Open(srcFilepath)
	if opErr != nil {
		return fmt.Errorf("failed to open file: %w", opErr)
	}

	defer srcFile.Close()

	dstFile, crErr := fs.Create(dstFilepath)
	if crErr != nil {
		return fmt.Errorf("failed to create file: %w", crErr)
	}

	defer dstFile.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err() //nolint:wrapcheck //wrapping adds nothing here
		default:
		}

		n, rErr := srcFile.Read(buf)
		if rErr != nil && !errors.Is(rErr, io.EOF) {
			return fmt.Errorf("failed to read file: %w", rErr)
		}

		if n > 0 {
			replaced := bytes.ReplaceAll(buf[:n], []byte(c.From), []byte(c.To))
			if _, wErr := dstFile.Write(replaced); wErr != nil {
				return fmt.Errorf("failed to write file: %w", wErr)
			}
		}

		if errors.Is(rErr, io.EOF) {
			break
		}
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}
