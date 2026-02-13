package main

import (
	"fmt"
	"net/url"
	"os"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type dummyConf map[string]any

func (*dummyConf) ValidPartner() error { return nil }

type transferContext[T protocol.PartnerConfig] struct {
	partnerCreds, accountCreds []api.OutCred
	realAddr                   string
	partnerConf                T
}

func getTransferContext[confType protocol.PartnerConfig](partner *api.OutPartner,
	account *api.OutRemoteAccount, restAddr string, insecure bool,
) (*transferContext[confType], error) {
	restPath, err := url.JoinPath(restAddr, "/api/partners", partner.Name, "credentials")
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	partnerCreds, err := getCreds(partner.Credentials, restPath, insecure)
	if err != nil {
		return nil, fmt.Errorf("could not get partner %s credentials: %w", partner.Name, err)
	}

	if restPath, err = url.JoinPath(restAddr, "/api/partners", partner.Name,
		"accounts", account.Login, "credentials"); err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	accountCreds, err := getCreds(account.Credentials, restPath, insecure)
	if err != nil {
		return nil, fmt.Errorf("could not get account %s credentials: %w", account.Login, err)
	}

	partnerConf := *new(confType)
	if err = utils.JSONConvert(partner.ProtoConfig, partnerConf); err != nil {
		return nil, fmt.Errorf("failed to parse WebDAV partner protocol configuration: %w", err)
	}

	realAddr, err := getRealAddress(partner.Address, restPath, insecure)
	if err != nil {
		return nil, fmt.Errorf("could not get address for partner %s: %w", partner.Name, err)
	}

	return &transferContext[confType]{
		partnerCreds: partnerCreds,
		accountCreds: accountCreds,
		realAddr:     realAddr,
		partnerConf:  partnerConf,
	}, nil
}

func getPassword(creds []api.OutCred) string {
	for _, cred := range creds {
		if cred.Type == auth.Password {
			return cred.Value
		}
	}

	return ""
}

type dirReader interface {
	ReadDir(string) ([]os.FileInfo, error)
}

func list(cli dirReader, rule *api.OutRule, pattern string) ([]string, error) {
	dirPattern := path.Dir(pattern)
	filePattern := path.Base(pattern)

	fileInfos, listErr := cli.ReadDir(path.Join(rule.RemoteDir, dirPattern))
	if listErr != nil {
		return nil, fmt.Errorf("failed to list files: %w", listErr)
	}

	res := []string{}

	for _, fi := range fileInfos {
		if !fi.IsDir() {
			ok, err := path.Match(filePattern, fi.Name())
			if err != nil {
				return nil, fmt.Errorf("bad pattern %q: %w", filePattern, err)
			}

			if ok {
				res = append(res, path.Join(dirPattern, fi.Name()))
			}
		}
	}

	return res, nil
}
