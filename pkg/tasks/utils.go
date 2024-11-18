package tasks

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func mapToStr(m map[string]string) string {
	args := make([]string, 0, len(m))
	for k, v := range m {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	return "{" + strings.Join(args, ", ") + "}"
}

type jsonDuration time.Duration

func (j *jsonDuration) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote duration: %w", err)
	}

	dur, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	*j = jsonDuration(dur)

	return nil
}

type jsonBool bool

func (j *jsonBool) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote bool: %w", err)
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		return fmt.Errorf("failed to parse bool: %w", err)
	}

	*j = jsonBool(b)

	return nil
}
