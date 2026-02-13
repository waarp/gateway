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
	case "ftp", "ftps":
		return &ftpClient{}, nil
	case "r66", "r66-tls":
		return &r66Client{}, nil
	case "webdav", "webdav-tls":
		return &webdavClient{}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol %q: %w", protocol, errUnknownProtocol)
	}
}
