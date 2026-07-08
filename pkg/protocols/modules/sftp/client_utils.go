package sftp

import (
	"slices"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// algorithmsForKeyFormat returns the supported signature algorithms for a given
// public key format (PublicKey.Type), in order of preference. See RFC 8332,
// Section 2. See also the note in sendKexInit on backwards compatibility.
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

func setDefaultClientAlgos(sshConf *ssh.ClientConfig) {
	if len(sshConf.KeyExchanges) == 0 {
		sshConf.KeyExchanges = ValidKeyExchanges.ClientDefaults()
	}

	if len(sshConf.Ciphers) == 0 {
		sshConf.Ciphers = ValidCiphers.ClientDefaults()
	}

	if len(sshConf.MACs) == 0 {
		sshConf.MACs = ValidMACs.ClientDefaults()
	}
}

func makePartnerHostKeys(logger *log.Logger, ctx *model.TransferContext,
) ([]ssh.PublicKey, []string, *pipeline.Error) {
	var (
		hostKeys []ssh.PublicKey
		algos    []string
	)

	partner := ctx.RemoteAgent.Name
	creds := ctx.RemoteAgentCreds

	for _, cred := range creds {
		key, err := ParseAuthorizedKey(cred.Value)
		if err != nil {
			logger.Warningf("Failed to parse the SFTP partner %q's hostkey %q: %v",
				partner, cred.Name, err)

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
		logger.Errorf("No valid hostkey found for partner %q", partner)

		return nil, nil, pipeline.NewErrorf(types.TeInternal,
			"no valid hostkey found for partner %q", partner)
	}

	return hostKeys, algos, nil
}

func makeClientAuthMethods(creds model.Credentials) []ssh.AuthMethod {
	var (
		signers  []ssh.Signer
		auths    []ssh.AuthMethod
		password string
	)

	for _, c := range creds {
		switch c.Type {
		case auth.Password:
			password = c.Value
		case AuthSSHPrivateKey:
			signer, err := ParsePrivateKey(c.Value)
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

func makeSSHClientConfig(logger *log.Logger, ctx *model.TransferContext, sshConfig *ssh.Config,
) (*ssh.ClientConfig, *pipeline.Error) {
	hostKeys, algos, err := makePartnerHostKeys(logger, ctx)
	if err != nil {
		return nil, err
	}

	authMethods := makeClientAuthMethods(ctx.RemoteAccountCreds)

	certChecker := &ssh.CertChecker{
		IsHostAuthority: isHostAuthority(ctx, logger),
		HostKeyFallback: makeFixedHostKeys(hostKeys),
	}

	clientConf := &ssh.ClientConfig{
		Config:            *sshConfig,
		User:              ctx.RemoteAccount.Login,
		Auth:              authMethods,
		HostKeyCallback:   certChecker.CheckHostKey,
		HostKeyAlgorithms: algos,
	}

	setDefaultClientAlgos(clientConf)

	return clientConf, nil
}
