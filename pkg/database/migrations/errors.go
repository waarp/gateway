package migrations

import "fmt"

func errUnknownEngine(dial string) error {
	//nolint:goerr113 //this is a base error
	return fmt.Errorf("unknown migration dialect engine '%s'", dial)
}
