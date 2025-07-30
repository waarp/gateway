package tasks

import (
	"encoding/json"
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

type jsonDuration struct{ time.Duration }

func (j *jsonDuration) UnmarshalJSON(bytes []byte) error {
	var str string
	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}

	dur, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	j.Duration = dur

	return nil
}

type jsonBool bool

func (j *jsonBool) UnmarshalJSON(bytes []byte) error {
	var str string
	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		return fmt.Errorf("failed to parse bool: %w", err)
	}

	*j = jsonBool(b)

	return nil
}

type jsonInt int64

func (j *jsonInt) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote int: %w", err)
	}

	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse int: %w", err)
	}

	*j = jsonInt(i)

	return nil
}

type jsonFloat float64

func (j *jsonFloat) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote float: %w", err)
	}

	i, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("failed to parse float: %w", err)
	}

	*j = jsonFloat(i)

	return nil
}

type jsonMap map[string]any

func (j *jsonMap) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote map: %w", err)
	}

	var m map[string]any
	if err = json.Unmarshal([]byte(str), &m); err != nil {
		return fmt.Errorf("failed to parse map: %w", err)
	}

	*j = m

	return nil
}
