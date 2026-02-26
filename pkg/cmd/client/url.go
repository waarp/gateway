package wg

import (
	"errors"
	"net/url"
	"strings"
)

//nolint:gochecknoglobals //global vars are required here
var (
	addr       url.URL
	insecure   bool
	addrEnvVar = "WAARP_GATEWAY_ADDRESS"
)

type AddrOpt struct{}

func (*AddrOpt) UnmarshalFlag(value string) error {
	if value == "" {
		return errors.New("the address flags '-a' is missing") //nolint:err113 // too specific base error
	}

	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}

	parsedURL, parsErr := url.ParseRequestURI(value)
	if parsErr != nil {
		var uErr *url.Error

		errors.As(parsErr, &uErr)

		return uErr.Err
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

	addr = *parsedURL

	return nil
}

type InsecureOpt func(string)

func SetInsecureFlag(value string) {
	insecure = strings.TrimSpace(value) != ""
}
