package main

import (
	"fmt"
	"net/http"

	"github.com/studio-b12/gowebdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	gwwd "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
)

type webdavClient struct {
	client *gowebdav.Client
}

func (w *webdavClient) Connect(partner *api.OutPartner, account *api.OutRemoteAccount,
	restAddr string, insecure bool,
) error {
	t, err := getTransferContext[*gwwd.PartnerConfigTLS](partner, account, restAddr, insecure)
	if err != nil {
		return err
	}

	accountPswd := getPassword(t.accountCreds)

	scheme := "http://"
	if partner.Protocol == gwwd.WebdavTLS {
		scheme = "https://"
	}

	w.client = gowebdav.NewClient(scheme+t.realAddr, account.Login, accountPswd)

	if partner.Protocol == gwwd.WebdavTLS {
		tlsConfig := getTLSConf(t.realAddr, t.partnerConf.MinTLSVersion, t.partnerCreds, t.accountCreds, nil)
		w.client.SetTransport(&http.Transport{TLSClientConfig: tlsConfig})
	}

	if err = w.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to WebDAV server: %w", err)
	}

	return nil
}

func (w *webdavClient) List(rule *api.OutRule, pattern string) ([]string, error) {
	return list(w.client, rule, pattern)
}

func (w *webdavClient) Close() error {
	return nil
}
