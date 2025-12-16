package main

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

type Client interface {
	Connect(partner *api.OutPartner, account *api.OutRemoteAccount, addr string, insecure bool) error
	List(rule *api.OutRule, pattern string) ([]string, error)
	Close() error
}

func newClient(protocol string) (Client, error) {
	switch protocol {
	case "sftp":
		return &sftpClient{}, nil
	case "ftp":
		return &ftpClient{}, nil
	case "ftps":
		return &ftpClient{}, nil
	case "r66":
		return &r66Client{}, nil
	case "r66-tls":
		return &r66Client{}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol %q: %w", protocol, errUnknownProtocol)
	}
}
