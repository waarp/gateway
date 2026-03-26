package rtn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	defaultRTNEventBuffer = 32
	defaultRTNErrorBuffer = 16
	defaultReconnectDelay = 3 * time.Second

	RTNProviderStateStopped  = "STOPPED"
	RTNProviderStateStarting = "STARTING"
	RTNProviderStateRunning  = "RUNNING"
	RTNProviderStateDegraded = "DEGRADED"
	RTNProviderStateFailed   = "FAILED"
)

var (
	errRTNProviderMissingName     = errors.New("the RTN WSS provider name is missing")
	errRTNProviderMissingEndpoint = errors.New("the RTN WSS provider endpoint is missing")
	errRTNProviderInvalidEndpoint = errors.New("the RTN WSS provider endpoint must use ws or wss")
)

// WSSProvider implements the RTN provider contract over WebSocket/WSS.
type WSSProvider struct {
	name     string
	endpoint string
	enabled  bool
	events   chan RawEvent
	errors   chan error

	state  utils.State
	cancel context.CancelFunc
	conn   *websocket.Conn
	mutex  sync.Mutex
}

// NewWSSProvider creates a new WSS RTN provider.
func NewWSSProvider(name, endpoint string, enabled bool) *WSSProvider {
	return &WSSProvider{
		name:     strings.TrimSpace(name),
		endpoint: strings.TrimSpace(endpoint),
		enabled:  enabled,
		events:   make(chan RawEvent, defaultRTNEventBuffer),
		errors:   make(chan error, defaultRTNErrorBuffer),
		state:    utils.NewState(utils.StateOffline, ""),
	}
}

// Start starts the WSS provider and its reconnect loop.
func (p *WSSProvider) Start(ctx context.Context) error {
	if p.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := p.validate(); err != nil {
		p.state.Set(utils.StateError, err.Error())
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.state.Set(utils.StateRunning, "")

	go p.reconnectLoop(runCtx)

	return nil
}

// Stop stops the WSS provider and closes the active connection.
func (p *WSSProvider) Stop(context.Context) error {
	if !p.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if p.cancel != nil {
		p.cancel()
	}

	p.closeConn()
	p.state.Set(utils.StateOffline, "")

	return nil
}

// State returns the current provider state.
func (p *WSSProvider) State() (state, reason string) {
	code, reason := p.state.Get()

	switch code {
	case utils.StateRunning:
		return RTNProviderStateRunning, reason
	case utils.StateError:
		if reason == "" {
			return RTNProviderStateFailed, reason
		}

		return RTNProviderStateDegraded, reason
	default:
		return RTNProviderStateStopped, reason
	}
}

// Events returns the normalized RTN event stream.
func (p *WSSProvider) Events() <-chan RawEvent { return p.events }

// Errors returns the provider error stream.
func (p *WSSProvider) Errors() <-chan error { return p.errors }

func (p *WSSProvider) validate() error {
	if p.name == "" {
		return errRTNProviderMissingName
	}

	if p.endpoint == "" {
		return errRTNProviderMissingEndpoint
	}

	endpoint, err := url.Parse(p.endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse the RTN WSS endpoint: %w", err)
	}

	if endpoint.Scheme != "ws" && endpoint.Scheme != "wss" {
		return errRTNProviderInvalidEndpoint
	}

	return nil
}

func (p *WSSProvider) reconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := p.connectAndRead(ctx); err != nil {
			p.state.Set(utils.StateError, err.Error())
			p.pushError(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(defaultReconnectDelay):
		}
	}
}

func (p *WSSProvider) connectAndRead(ctx context.Context) error {
	config, err := websocket.NewConfig(p.endpoint, "http://localhost/")
	if err != nil {
		return fmt.Errorf("failed to build the RTN WSS configuration: %w", err)
	}

	conn, err := websocket.DialConfig(config)
	if err != nil {
		return fmt.Errorf("failed to connect the RTN WSS provider: %w", err)
	}

	p.setConn(conn)
	defer p.closeConn()

	p.state.Set(utils.StateRunning, "")

	return p.readLoop(ctx, conn)
}

func (p *WSSProvider) readLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		var rawMessage []byte
		if err := websocket.Message.Receive(conn, &rawMessage); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("failed to read an RTN WSS message: %w", err)
		}

		rawEvent, err := p.normalizeMessage(rawMessage)
		if err != nil {
			p.pushError(err)
			continue
		}

		select {
		case <-ctx.Done():
			return nil
		case p.events <- rawEvent:
		}
	}
}

func (p *WSSProvider) normalizeMessage(raw []byte) (RawEvent, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RawEvent{}, fmt.Errorf("failed to decode the RTN WSS payload: %w", err)
	}

	event := RawEvent{
		Source:     p.name,
		Payload:    raw,
		Metadata:   payload,
		ReceivedAt: time.Now().UTC(),
	}

	if source, ok := payload["source"].(string); ok && strings.TrimSpace(source) != "" {
		event.Source = strings.TrimSpace(source)
	}

	if eventID, ok := payload["eventID"].(string); ok {
		event.EventID = strings.TrimSpace(eventID)
	} else {
		alternateEventID, alternateOK := payload["eventId"].(string)
		if alternateOK {
			event.EventID = strings.TrimSpace(alternateEventID)
		}
	}

	if correlationID, ok := payload["correlationID"].(string); ok {
		event.CorrelationID = strings.TrimSpace(correlationID)
	} else {
		alternateCorrelationID, alternateOK := payload["correlationId"].(string)
		if alternateOK {
			event.CorrelationID = strings.TrimSpace(alternateCorrelationID)
		}
	}

	if receivedAtRaw, ok := payload["receivedAt"].(string); ok && strings.TrimSpace(receivedAtRaw) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(receivedAtRaw))
		if err != nil {
			return RawEvent{}, fmt.Errorf("failed to parse RTN receivedAt: %w", err)
		}

		event.ReceivedAt = parsed.UTC()
	}

	return event, nil
}

func (p *WSSProvider) pushError(err error) {
	select {
	case p.errors <- err:
	default:
	}
}

func (p *WSSProvider) setConn(conn *websocket.Conn) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.conn = conn
}

func (p *WSSProvider) closeConn() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}
