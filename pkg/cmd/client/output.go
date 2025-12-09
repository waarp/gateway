package wg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

var ErrUnknownOutputFormat = errors.New("unknown output format")

//nolint:lll //tags can be long for flags
type OutputFormat struct {
	Format string `long:"format" description:"The command's output format" choice:"human" choice:"json" choice:"yaml" default:"human"`
}

func outputObject[T any](w io.Writer, obj T, formatting *OutputFormat,
	humanReadable func(io.Writer, T) error,
) error {
	switch formatting.Format {
	case "human", "":
		return humanReadable(w, obj)
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(obj); err != nil {
			return fmt.Errorf("failed to encode JSON object: %w", err)
		}
	case "yaml":
		const indent = 2
		encoder := yaml.NewEncoder(w)
		encoder.SetIndent(indent)

		if err := encoder.Encode(obj); err != nil {
			return fmt.Errorf("failed to encode YAML object: %w", err)
		}
	default:
		return fmt.Errorf("%w %q", ErrUnknownOutputFormat, formatting)
	}

	return nil
}
