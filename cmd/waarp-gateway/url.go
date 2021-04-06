package main

import (
	"fmt"
	"net/url"
	"strings"
)

type addrOpt struct {
	Address  gwAddr `short:"a" long:"address" description:"The address of the gateway" env:"WAARP_GATEWAY_ADDRESS"`
	Insecure bool   `short:"i" long:"insecure" description:"Skip certificate verification" env:"WAARP_GATEWAY_INSECURE"`
}

type gwAddr struct{}

func (*gwAddr) UnmarshalFlag(value string) error {
	if value == "" {
		return fmt.Errorf("the address flags '-a' is missing")
	}

	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return err.(*url.Error).Err
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

type listOptions struct {
	Limit  int `short:"l" long:"limit" description:"Max number of returned entries" default:"20"`
	Offset int `short:"o" long:"offset" description:"Index of the first returned entry" default:"0"`
}

func agentListURL(path string, s *listOptions, sort string, protos []string) {
	addr.Path = path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	for _, proto := range protos {
		query.Add("protocol", proto)
	}
	addr.RawQuery = query.Encode()
}

func listURL(s *listOptions, sort string) {
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)
	addr.RawQuery = query.Encode()
}
