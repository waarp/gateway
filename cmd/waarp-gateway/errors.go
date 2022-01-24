package main

import "errors"

var (
	errInvalidDate = errors.New("invalid date")
	errBadArgs     = errors.New("bad arguments")
	errNothingToDo = errors.New("nothing to do")
)
