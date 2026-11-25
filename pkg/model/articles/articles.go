// Package articles describes the articles of a file in the transfer
// information: the keys that hold them, the lengths of the articles, which
// a PeSIT receiver keeps and a task that changes the records of the file
// keeps true, and the values of the format and of the separator.
package articles

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
)

// The keys of the transfer information that describe the articles of the
// file: their lengths, their format and the line end at which a text file is
// cut into articles before an emission. Protocol modules that need them can
// declare them among their reserved keys. The tasks that change the records of
// a file read them here.
const (
	LengthsKey   = "__articlesLengths__"
	FormatKey    = "__articlesFormat__"
	SeparatorKey = "__articlesSeparator__"
)

// The values of the format of the articles and of the line end at which a text
// file is cut into articles before an emission.
const (
	FormatFixed    = "Fixed"
	FormatVariable = "Variable"
	SeparatorLF    = "LF"
	SeparatorCRLF  = "CRLF"
)

// ErrBadLengths is returned for lengths that cannot be read.
var ErrBadLengths = errors.New("invalid articles lengths")

// run is a run of consecutive articles of the same length L.
type run struct {
	L uint16 `json:"l"`
	C uint64 `json:"c"`
}

//nolint:wrapcheck //wrapping adds nothing here
func (l *run) MarshalJSON() ([]byte, error) {
	if l.C == 1 {
		return json.Marshal(l.L)
	}

	type lengths run
	ls := lengths(*l)

	return json.Marshal(&ls)
}

//nolint:wrapcheck //wrapping adds nothing here
func (l *run) UnmarshalJSON(b []byte) error {
	type lengths run
	var ls lengths

	if err := json.Unmarshal(b, &ls); err != nil {
		var i uint16
		if json.Unmarshal(b, &i) != nil {
			return err
		}

		ls.L = i
		ls.C = 1
	}

	if ls.C == 0 {
		return fmt.Errorf("%w: count cannot be zero", ErrBadLengths)
	}

	l.L = ls.L
	l.C = ls.C

	return nil
}

// Lengths is the length of each article of a file, stored
// run-length encoded: a file made of many articles of the same size takes a
// few bytes in the transfer info rather than one number per article.
type Lengths []run

// NewLengths returns the given lengths in their compact form.
func NewLengths(lengths ...uint16) *Lengths {
	l := &Lengths{}
	for _, length := range lengths {
		l.Add(length)
	}

	return l
}

// Add appends the length of one more article.
func (l *Lengths) Add(length uint16) {
	l.addRun(length, 1)
}

func (l *Lengths) addRun(length uint16, count uint64) {
	if last := len(*l) - 1; last >= 0 && (*l)[last].L == length {
		(*l)[last].C += count

		return
	}

	*l = append(*l, run{L: length, C: count})
}

// Len returns the number of articles.
func (l *Lengths) Len() int {
	if l == nil {
		return 0
	}

	var n uint64
	for _, run := range *l {
		n += run.C
	}

	return int(n)
}

// Max returns the length of the longest article, 0 when there is none.
func (l *Lengths) Max() uint16 {
	var longest uint16

	if l != nil {
		for _, run := range *l {
			longest = max(longest, run.L)
		}
	}

	return longest
}

// Uniform tells whether the articles all have the same length, as articles of
// a fixed length do; none and one do.
func (l *Lengths) Uniform() bool {
	if l != nil {
		for _, run := range *l {
			if run.L != (*l)[0].L {
				return false
			}
		}
	}

	return true
}

// All iterates over the length of each article, in order.
func (l *Lengths) All() iter.Seq[uint16] {
	return func(yield func(uint16) bool) {
		if l == nil {
			return
		}

		for _, run := range *l {
			for range run.C {
				if !yield(run.L) {
					return
				}
			}
		}
	}
}

func (l *Lengths) String() string {
	b, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	return string(b)
}
