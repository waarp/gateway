package tasks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type jsonDuration struct{ time.Duration }

func (j *jsonDuration) IsZero() bool { return j.Duration == 0 }
func (j *jsonDuration) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote duration: %w", err)
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

type jsonList []string

func (j *jsonList) UnmarshalJSON(bytes []byte) error {
	str, err := strconv.Unquote(string(bytes))
	if err != nil {
		return fmt.Errorf("failed to unquote list: %w", err)
	}

	l := strings.Split(str, ",")
	for i, s := range l {
		l[i] = strings.TrimSpace(s)
	}

	*j = l

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

type jsonObject map[string]any

func (j *jsonObject) UnmarshalJSON(bytes []byte) error {
	str, uqErr := strconv.Unquote(string(bytes))
	if uqErr != nil {
		return fmt.Errorf("failed to unquote object: %w", uqErr)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(str), &m); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	*j = m

	return nil
}
