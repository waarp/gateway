package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"slices"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	gwsftp "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

var errNoKeyFound = errors.New("no valid hostkey found")

type sftpClient struct {
	sshConfig  ssh.Config
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func (sc *sftpClient) Connect(partner *api.OutPartner, account *api.OutRemoteAccount, addr string, insecure bool,
) error {
	var err error
	var partnerCreds []api.OutCred
	var accountCreds []api.OutCred

	restPath, urlErr := url.JoinPath(addr, "/api/partners", partner.Name, "credentials")
	if urlErr != nil {
		return fmt.Errorf("failed to build URL: %w", urlErr)
	}

	if partnerCreds, err = getCreds(partner.Credentials, restPath, insecure); err != nil {
		return fmt.Errorf("could not get partner %s credentials: %w", partner.Name, err)
	}

	restPath, urlErr = url.JoinPath(addr, "/api/partners", partner.Name, "accounts", account.Login, "credentials")
	if urlErr != nil {
		return fmt.Errorf("failed to build URL: %w", urlErr)
	}

	if accountCreds, err = getCreds(account.Credentials, restPath, insecure); err != nil {
		return fmt.Errorf("could not get account %s credentials: %w", account.Login, err)
	}

	if netErr := sc.openSSHConn(partner, account, partnerCreds, accountCreds); netErr != nil {
		return netErr
	}

	return sc.startSFTPSession()
}

func (sc *sftpClient) List(rule *api.OutRule, pattern string) ([]string, error) {
	res, listErr := sc.sftpClient.Glob(path.Join(rule.RemoteDir, pattern))
	if listErr != nil {
		return nil, fmt.Errorf("failed to list files: %w", listErr)
	}

	return res, nil
}

func (sc *sftpClient) Close() error {
	sc.sftpClient.Close()
	sc.sshClient.Close()

	return nil
}

func (sc *sftpClient) makeSSHClientConfig(account *api.OutRemoteAccount, remoteAgentCreds,
	remoteAccountCreds []api.OutCred,
) (*ssh.ClientConfig, error) {
	hostKeys, algos, err := makePartnerHostKeys(remoteAgentCreds)
	if err != nil {
		return nil, err
	}

	authMethods := makeClientAuthMethods(remoteAccountCreds)

	certChecker := &ssh.CertChecker{
		HostKeyFallback: makeFixedHostKeys(hostKeys),
	}

	sshConf := &ssh.ClientConfig{
		Config:            sc.sshConfig,
		User:              account.Login,
		Auth:              authMethods,
		HostKeyCallback:   certChecker.CheckHostKey,
		HostKeyAlgorithms: algos,
	}

	// setDefaultClientAlgos(sshConf)

	return sshConf, nil
}

func (sc *sftpClient) openSSHConn(partner *api.OutPartner, account *api.OutRemoteAccount, remoteAgentCreds,
	remoteAccountCreds []api.OutCred,
) error {
	sshClientConf, confErr := sc.makeSSHClientConfig(account, remoteAgentCreds, remoteAccountCreds)
	if confErr != nil {
		return confErr
	}

	/*
		addr := conf.GetRealAddress(c.pip.TransCtx.RemoteAgent.Address.Host,
			utils.FormatUint(c.pip.TransCtx.RemoteAgent.Address.Port))
	*/

	conn, dialErr := net.Dial("tcp", partner.Address)
	if dialErr != nil {
		return fmt.Errorf("failed to connect to the SFTP partner: %w", dialErr)
	}

	sshConn, chans, reqs, sshErr := ssh.NewClientConn(conn, partner.Address, sshClientConf)
	if sshErr != nil {
		return fmt.Errorf("failed to start the SSH session: %w", sshErr)
	}

	sc.sshClient = ssh.NewClient(sshConn, chans, reqs)

	return nil
}

func (sc *sftpClient) startSFTPSession() error {
	var opts []sftp.ClientOption

	var sftpErr error

	sc.sftpClient, sftpErr = sftp.NewClient(sc.sshClient, opts...)
	if sftpErr != nil {
		return fmt.Errorf("failed to start SFTP session: %w", sftpErr)
	}

	return nil
}

func algorithmsForKeyFormat(keyFormat string) []string {
	switch keyFormat {
	case ssh.KeyAlgoRSA:
		return []string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA}
	case ssh.CertAlgoRSAv01:
		return []string{ssh.CertAlgoRSASHA256v01, ssh.CertAlgoRSASHA512v01, ssh.CertAlgoRSAv01}
	default:
		return []string{keyFormat}
	}
}

func makePartnerHostKeys(creds []api.OutCred) ([]ssh.PublicKey, []string, error) {
	var (
		hostKeys []ssh.PublicKey
		algos    []string
	)

	for _, cred := range creds {
		key, err := gwsftp.ParseAuthorizedKey(cred.Value)
		if err != nil {
			// TODO  log("Failed to parse the SFTP partner hostkey %q: %v",cred.Name, err)
			continue
		}

		hostKeys = append(hostKeys, key)

		for _, newAlgo := range algorithmsForKeyFormat(key.Type()) {
			if !slices.Contains(algos, newAlgo) {
				algos = append(algos, newAlgo)
			}
		}
	}

	if len(hostKeys) == 0 {
		return nil, nil, errNoKeyFound
	}

	return hostKeys, algos, nil
}

func makeClientAuthMethods(creds []api.OutCred) []ssh.AuthMethod {
	var (
		signers  []ssh.Signer
		auths    []ssh.AuthMethod
		password string
	)

	for _, c := range creds {
		switch c.Type {
		case auth.Password:
			password = c.Value
		case gwsftp.AuthSSHPrivateKey:
			signer, err := gwsftp.ParsePrivateKey(c.Value)
			if err != nil {
				continue
			}

			signers = append(signers, signer)
		}
	}

	if len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}

	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	return auths
}

var (
	errSSHNoKey       = errors.New("no key found")
	errSSHKeyMismatch = errors.New("the SSH key does not match known keys")
)

type fixedHostKeys []ssh.PublicKey

func (f fixedHostKeys) check(_ string, _ net.Addr, remoteKey ssh.PublicKey) error {
	if len(f) == 0 {
		return fmt.Errorf("ssh: required host key was nil: %w", errSSHNoKey)
	}

	remoteBytes := remoteKey.Marshal()
	for _, key := range f {
		if bytes.Equal(remoteBytes, key.Marshal()) {
			return nil
		}
	}

	return fmt.Errorf("ssh: host key mismatch: %w", errSSHKeyMismatch)
}

func makeFixedHostKeys(keys []ssh.PublicKey) ssh.HostKeyCallback {
	hk := fixedHostKeys(keys)

	return hk.check
}
