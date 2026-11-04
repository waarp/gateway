package wg

import "errors"

var (
	errInvalidDate   = errors.New("invalid date")
	errMissingTarget = errors.New("the --target option is required when no other option names the transfers to cancel")
	errBadArgs       = errors.New("bad arguments")
	errNothingToDo   = errors.New("nothing to do")
)
