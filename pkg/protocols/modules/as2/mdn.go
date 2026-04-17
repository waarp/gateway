package as2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/lib/as2"
	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type unknownMessageIDError string

func (e unknownMessageIDError) Error() string {
	return fmt.Sprintf("no transfer found for message ID %s", string(e))
}

type asyncStore struct {
	m *xsync.Map[string, *utils.NullableChan[*as2.MDN]]
}

func newAsyncStore() *asyncStore {
	return &asyncStore{m: xsync.NewMap[string, *utils.NullableChan[*as2.MDN]]()}
}

func (a *asyncStore) Record(_ context.Context, msgID string, mdn *as2.MDN, _ string, _ []byte) error {
	ch, ok := a.m.Load(msgID)
	if !ok {
		return unknownMessageIDError(msgID)
	}

	ch.Send(mdn)

	return nil
}

//nolint:nilnil //nil here is required by the interface's definition
func (a *asyncStore) GetMDN(context.Context, string) (*as2.MDN, string, []byte, error) {
	return nil, "", nil, nil
}

func (a *asyncStore) asyncListen(pip *pipeline.Pipeline, _ *protoutils.TraceDialer,
) (net.Listener, error) {
	partner := pip.TransCtx.RemoteAgent

	var partConf partnerProtoConfigTLS
	if err := utils.JSONConvert(partner.ProtoConfig, &partConf); err != nil {
		return nil, fmt.Errorf("failed to parse partner proto config: %w", err)
	}

	handler := as2.NewMDNHandler(nil, a, nil)
	list, listErr := net.Listen("tcp", partConf.AsyncMDNAddress)
	if listErr != nil {
		return nil, fmt.Errorf("failed to listen on %q: %w", partConf.AsyncMDNAddress, listErr)
	}

	if partner.Protocol == AS2TLS {
		tlsConfig, err := asyncTLSConfig(pip, &partConf)
		if err != nil {
			return nil, fmt.Errorf("failed to get TLS config for async MDN: %w", err)
		}

		list = tls.NewListener(list, tlsConfig)
	}

	//nolint:errcheck //we don't care about the error here
	go http.Serve(list, handler)

	return list, nil
}

func asyncTLSConfig(pip *pipeline.Pipeline, partConf *partnerProtoConfigTLS) (*tls.Config, error) {
	conf, err := protoutils.GetClientTLSConfig(pip.TransCtx, logging.Discard(), partConf.MinTLSVersion)
	if err != nil {
		return nil, err
	}

	conf.ServerName = partConf.AsyncMDNAddress

	return conf, nil
}
