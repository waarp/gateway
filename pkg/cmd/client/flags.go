package wg

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/jessevdk/go-flags"
)

type textFile struct{ flags.Filename }

func (f *textFile) IsZero() bool { return f.Filename == "" }

func (f *textFile) UnmarshalFlag(value string) error {
	f.Filename = flags.Filename(value)

	return nil
}

func (f *textFile) MarshalJSON() ([]byte, error) {
	if f.Filename == "" {
		return []byte("null"), nil
	}

	name := string(f.Filename)
	cont, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", name, err)
	}

	return []byte(strconv.Quote(string(cont))), nil
}

type textFileOrValue struct{ textFile }

func (f *textFileOrValue) MarshalJSON() ([]byte, error) {
	if b, err := f.textFile.MarshalJSON(); err == nil {
		return b, nil
	}

	return []byte(strconv.Quote(string(f.Filename))), nil
}

type jsonObject map[string]any

//nolint:wrapcheck //function is already a wrapper, best not wrap errors as well
func (j *jsonObject) UnmarshalFlag(value string) error {
	if value == "" {
		return nil
	}

	return json.Unmarshal([]byte(value), j)
}

type jsonObjects []jsonObject

//nolint:wrapcheck //function is already a wrapper, best not wrap errors as well
func (js *jsonObjects) MarshalJSON() ([]byte, error) {
	switch {
	case js == nil:
		return []byte("null"), nil
	case len(*js) == 0:
		return []byte("[]"), nil
	default:
		return json.Marshal(*js)
	}
}

//nolint:wrapcheck //function is already a wrapper, best not wrap errors as well
func (js *jsonObjects) UnmarshalFlag(value string) error {
	if value == "" {
		return nil
	}

	j := jsonObject{}
	if err := j.UnmarshalFlag(value); err != nil {
		return err
	}

	*js = append(*js, j)

	return nil
}
