package ebics

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig        = errors.New("invalid EBICS configuration")
	ErrUnsupportedTransport = errors.New("unsupported EBICS transport")
	ErrUnsupportedVersion   = errors.New("unsupported EBICS protocol version")
	ErrNotImplemented       = errors.New("EBICS feature not implemented yet")
)

func wrapConfigError(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
}
