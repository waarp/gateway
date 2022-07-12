package wg

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

//nolint:gochecknoglobals //global vars are required here
var (
	addr     *url.URL
	insecure bool
)

type AddrOpt struct{}

func (*AddrOpt) UnmarshalFlag(value string) error {
	if value == "" {
		return fmt.Errorf("the address flags '-a' is missing") //nolint:goerr113 // too specific base error
	}

	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		var err2 *url.Error

		errors.As(err, &err2)

		return err2.Err
	}

	if _, hasPwd := parsedURL.User.Password(); !hasPwd {
		user := parsedURL.User.Username()
		if user == "" {
			var err error
			if user, err = promptUser(); err != nil {
				return err
			}
		}

		pwd, err := promptPassword()
		if err != nil {
			return err
		}

		parsedURL.User = url.UserPassword(user, pwd)
	}

	addr = parsedURL

	return nil
}

type InsecureOpt func(string)

func SetInsecureFlag(value string) {
	insecure = strings.TrimSpace(value) != ""
}
