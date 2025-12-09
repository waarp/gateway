package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// JSONConvert takes a source object and converts it to the destination object
// using JSON to map the object fields. Useful to convert between structs and
// maps (and vice-versa).
func JSONConvert(from, to any) error {
	var (
		buf     bytes.Buffer
		encoder = json.NewEncoder(&buf)
		decoder = json.NewDecoder(&buf)
	)

	if err := encoder.Encode(from); err != nil {
		return fmt.Errorf("failed to re-encode the JSON object: %w", err)
	}

	if err := decoder.Decode(to); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

type jsonBody struct {
	Obj  map[string]any
	buf  bytes.Buffer
	once sync.Once
}

func ToJSONBody(obj map[string]any) io.Reader {
	return &jsonBody{Obj: obj}
}

//nolint:wrapcheck //this is just a wrapper func, no need to wrap errors
func (j *jsonBody) Read(p []byte) (int, error) {
	var err error

	j.once.Do(func() {
		err = json.NewEncoder(&j.buf).Encode(j.Obj)
	})

	if err != nil {
		return 0, err
	}

	return j.buf.Read(p)
}

func MustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(b)
}
