package file

import (
	"encoding/json"
	"strconv"
	"time"
)

func (s *SNMPServer) IsZero() bool {
	return s.LocalUDPAddress == "" &&
		s.Community == "" &&
		!s.V3Only &&
		s.V3Username == "" &&
		s.V3AuthProtocol == "" &&
		s.V3AuthPassphrase == "" &&
		s.V3PrivacyProtocol == "" &&
		s.V3PrivacyPassphrase == ""
}

func (s SNMPConfig) IsZero() bool {
	return len(s.Monitors) == 0 && s.Server.IsZero()
}

func (e EmailConfig) IsZero() bool {
	return len(e.Credentials) == 0 && len(e.Templates) == 0
}

type Duration time.Duration

func (d Duration) IsZero() bool { return d == 0 }

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(time.Duration(d).String())), nil
}

//nolint:wrapcheck //wrapping adds nothing here
func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	*d = Duration(duration)

	return nil
}
