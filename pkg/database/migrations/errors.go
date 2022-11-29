package migrations

import (
	"errors"
	"fmt"
)

var ErrUnknownDialect = errors.New("unknown SQL dialect")

func errUnknownDialect(dial string) error {
	return fmt.Errorf("%w: %q", ErrUnknownDialect, dial)
}
