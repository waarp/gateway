package api

import (
	"encoding/json"
	"strconv"
	"time"
)

// Duration is a time.Duration that marshals to/from a JSON string in the format
// accepted by time.ParseDuration (e.g. "1h30m", "5m", "300ms").
type Duration time.Duration

func (d Duration) IsZero() bool { return d == 0 }

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(time.Duration(d).String())), nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	*d = Duration(dur)

	return nil
}
