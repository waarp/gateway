package wg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
)

type file string

func (f *file) Complete(match string) []flags.Completion {
	filename := flags.Filename("")

	return filename.Complete(match)
}

func (f *file) UnmarshalFlag(value string) error {
	if value == "" {
		return nil
	}

	cont, err := os.ReadFile(filepath.Clean(value))
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", value, err)
	}

	*f = file(cont)

	return nil
}

type fileOrValue string

func (f *fileOrValue) Complete(match string) []flags.Completion {
	filename := flags.Filename("")

	return filename.Complete(match)
}

func (f *fileOrValue) UnmarshalFlag(value string) error {
	if value == "" {
		return nil
	}

	if _, err := os.Stat(filepath.Clean(value)); err != nil {
		*f = fileOrValue(value)

		return nil
	}

	cont, err := os.ReadFile(filepath.Clean(value))
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", value, err)
	}

	*f = fileOrValue(cont)

	return nil
}

type jsonObject map[string]any

//nolint:wrapcheck //function is already a wrapper, best not wrap errors as well
func (j *jsonObject) UnmarshalFlag(value string) error {
	if value == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(value), j); err != nil {
		return err
	}

	return nil
}
