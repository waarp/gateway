package rtn

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/net/websocket"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type WSSNotifier struct {
	name     string
	endpoint string
}

func NewWSSNotifier(name, endpoint string) *WSSNotifier {
	return &WSSNotifier{
		name:     strings.TrimSpace(name),
		endpoint: strings.TrimSpace(endpoint),
	}
}

func (n *WSSNotifier) Publish(_ context.Context, payload []byte) error {
	if err := n.validate(); err != nil {
		return err
	}

	config, cfgErr := websocket.NewConfig(n.endpoint, "http://localhost/")
	if cfgErr != nil {
		return fmt.Errorf("failed to build the outbound RTN WSS configuration: %w", cfgErr)
	}

	conn, dialErr := websocket.DialConfig(config)
	if dialErr != nil {
		return fmt.Errorf("failed to connect the outbound RTN WSS provider: %w", dialErr)
	}
	defer func() { _ = conn.Close() }()

	sendErr := websocket.Message.Send(conn, payload)
	if sendErr != nil {
		return fmt.Errorf("failed to send the outbound RTN WSS payload: %w", sendErr)
	}

	return nil
}

func (n *WSSNotifier) validate() error {
	if n.name == "" {
		return errRTNProviderMissingName
	}
	if n.endpoint == "" {
		return errRTNProviderMissingEndpoint
	}

	if err := model.ValidateEbicsRTNOutboundEndpoint(n.endpoint); err != nil {
		return fmt.Errorf("failed to validate the outbound RTN WSS endpoint: %w", err)
	}

	return nil
}
