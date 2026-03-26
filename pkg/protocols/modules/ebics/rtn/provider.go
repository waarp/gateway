package rtn

import (
	"context"
	"time"
)

// RawEvent represents a transport-normalized RTN message ready for ingestion.
type RawEvent struct {
	Source        string
	EventID       string
	CorrelationID string
	Payload       []byte
	Metadata      map[string]any
	ReceivedAt    time.Time
}

// Provider defines the transport contract for RTN providers.
type Provider interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	State() (string, string)
	Events() <-chan RawEvent
	Errors() <-chan error
}
