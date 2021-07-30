package migrations

import "fmt"

func errUnknownEngine(dial string) error {
	return fmt.Errorf("unknown migration dialect engine '%s'", dial)
}
