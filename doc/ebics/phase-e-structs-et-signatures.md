# Phase E - Structs Go et signatures de methodes

## 1. Objet

Ce document fixe les declarations cibles de la `Phase E`.

Il couvre:

- `EbicsRTNEvent`;
- `EbicsRTNProvider`;
- les interfaces provider RTN;
- les runtimes d'ingestion et d'auto-pull;
- les DTO REST minimaux.

## 2. Conventions

Regles a conserver:

- coeur RTN decouple du transport;
- persistance durable et idempotente;
- compatibilite multi-SGBD via XORM;
- aucune logique metier client dans le coeur RTN;
- `WebSocket/WSS` comme provider de phase 1, pas comme coeur du modele.

## 3. Constantes model

## 3.1 `pkg/model/table_names.go`

Ajouts cibles:

```go
const (
	TableEbicsRTNEvents    = "ebics_rtn_events"
	TableEbicsRTNProviders = "ebics_rtn_providers"
)
```

## 3.2 `pkg/model/display_names.go`

Ajouts cibles:

```go
const (
	NameEbicsRTNEvent    = "ebics RTN event"
	NameEbicsRTNProvider = "ebics RTN provider"
)
```

## 4. Models `pkg/model`

## 4.1 `pkg/model/ebics_rtn_event.go`

### Struct cible

```go
package model

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type EbicsRTNEvent struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	Source            string        `xorm:"source"`
	EventID           string        `xorm:"event_id"`
	CorrelationID     string        `xorm:"correlation_id"`
	IdempotenceKey    string        `xorm:"idempotence_key"`
	EbicsHostID       sql.NullInt64 `xorm:"ebics_host_id"`
	EbicsSubscriberID sql.NullInt64 `xorm:"ebics_subscriber_id"`
	OrderTypeHint     string        `xorm:"order_type_hint"`
	ProfileID         string        `xorm:"profile_id"`
	Payload           string        `xorm:"payload"`
	Status            string        `xorm:"status"`
	Attempts          int           `xorm:"attempts"`
	NextRetryAt       time.Time     `xorm:"next_retry_at DATETIME(6) UTC"`
	ReceivedAt        time.Time     `xorm:"received_at DATETIME(6) UTC"`
	ProcessedAt       time.Time     `xorm:"processed_at DATETIME(6) UTC"`
	LastError         string        `xorm:"last_error"`
	CreatedAt         time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsRTNEvent) TableName() string
func (*EbicsRTNEvent) Appellation() string
func (e *EbicsRTNEvent) GetID() int64
func (e *EbicsRTNEvent) BeforeWrite(db database.Access) error
func (e *EbicsRTNEvent) AfterRead(database.ReadAccess) error
func (e *EbicsRTNEvent) AfterInsert(db database.Access) error
func (e *EbicsRTNEvent) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsRTNEventStatus(value string) error
func validateEbicsRTNEventSource(value string) error
func validateEbicsRTNEventCoherence(e *EbicsRTNEvent) error
```

## 4.2 `pkg/model/ebics_rtn_provider.go`

### Struct cible

```go
package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type EbicsRTNProvider struct {
	ID                int64  `xorm:"<- id AUTOINCR"`
	Owner             string `xorm:"owner"`
	Name              string `xorm:"name"`
	Transport         string `xorm:"transport"`
	Enabled           bool   `xorm:"enabled"`
	EbicsSubscriberID int64  `xorm:"ebics_subscriber_id"`
	Configuration     string `xorm:"configuration"`
	AutoPullPolicy    string `xorm:"auto_pull_policy"`
	LastConnectionAt  time.Time `xorm:"last_connection_at DATETIME(6) UTC"`
	LastError         string    `xorm:"last_error"`
	CreatedAt         time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsRTNProvider) TableName() string
func (*EbicsRTNProvider) Appellation() string
func (p *EbicsRTNProvider) GetID() int64
func (p *EbicsRTNProvider) BeforeWrite(db database.Access) error
func (p *EbicsRTNProvider) AfterRead(database.ReadAccess) error
func (p *EbicsRTNProvider) AfterInsert(db database.Access) error
func (p *EbicsRTNProvider) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsRTNTransport(value string) error
func validateEbicsRTNAutoPullPolicy(value string) error
```

## 5. Package `pkg/protocols/modules/ebics/rtn`

## 5.1 `provider.go`

### Structs et interfaces cibles

```go
package rtn

import "context"

type RawEvent struct {
	Source    string
	EventID   string
	Payload   []byte
	Metadata  map[string]any
	ReceivedAt string
}

type Provider interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	State() (string, string)
	Events() <-chan RawEvent
	Errors() <-chan error
}
```

## 5.2 `wss_provider.go`

### Struct cible

```go
package rtn

type WSSProvider struct {
	name      string
	endpoint  string
	enabled   bool
	events    chan RawEvent
	errors    chan error
}
```

### Signatures cibles

```go
func NewWSSProvider(name string, endpoint string) *WSSProvider
func (p *WSSProvider) Start(ctx context.Context) error
func (p *WSSProvider) Stop(ctx context.Context) error
func (p *WSSProvider) State() (string, string)
func (p *WSSProvider) Events() <-chan RawEvent
func (p *WSSProvider) Errors() <-chan error
```

### Helpers prives recommandes

```go
func (p *WSSProvider) reconnectLoop(ctx context.Context)
func (p *WSSProvider) readLoop(ctx context.Context)
func (p *WSSProvider) normalizeMessage(raw []byte) (RawEvent, error)
```

## 6. Runtime `pkg/protocols/modules/ebics/runtime`

## 6.1 `rtn_ingestion.go`

### Structs cibles

```go
package runtime

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/rtn"
)

type RTNIngestionResult struct {
	Event          *model.EbicsRTNEvent
	IsDuplicate    bool
	ProcessingPlan string
}
```

### Interfaces cibles

```go
type RTNEventStore interface {
	InsertRTNEvent(event *model.EbicsRTNEvent) error
	UpdateRTNEvent(event *model.EbicsRTNEvent) error
	GetRTNEventByIdempotenceKey(owner, key string) (*model.EbicsRTNEvent, error)
}
```

### Signatures cibles

```go
func IngestRTNEvent(
	owner string,
	raw rtn.RawEvent,
	idempotenceKey string,
	store RTNEventStore,
) (*RTNIngestionResult, error)

func ComputeRTNIdempotenceKey(raw rtn.RawEvent) (string, error)
```

## 6.2 `rtn_autopull.go`

### Structs cibles

```go
package runtime

type AutoPullPlan struct {
	Enabled      bool
	OrderType    string
	ProfileName  string
	CorrelationID string
	Reason       string
}
```

### Signatures cibles

```go
func BuildAutoPullPlan(event *model.EbicsRTNEvent, provider *model.EbicsRTNProvider) (*AutoPullPlan, error)
func CanAutoPull(event *model.EbicsRTNEvent, provider *model.EbicsRTNProvider) error
```

## 7. DTO API `pkg/admin/rest/api`

## 7.1 `ebics_rtn.go`

### Structs cibles

```go
type OutEbicsRTNEvent struct {
	ID                int64          `json:"id" yaml:"id"`
	Source            string         `json:"source" yaml:"source"`
	EventID           string         `json:"eventId,omitempty" yaml:"eventId,omitempty"`
	CorrelationID     string         `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	IdempotenceKey    string         `json:"idempotenceKey" yaml:"idempotenceKey"`
	OrderTypeHint     string         `json:"orderTypeHint,omitempty" yaml:"orderTypeHint,omitempty"`
	ProfileID         string         `json:"profileId,omitempty" yaml:"profileId,omitempty"`
	Status            string         `json:"status" yaml:"status"`
	Attempts          int            `json:"attempts" yaml:"attempts"`
	NextRetryAt       *time.Time     `json:"nextRetryAt,omitempty" yaml:"nextRetryAt,omitempty"`
	ReceivedAt        time.Time      `json:"receivedAt" yaml:"receivedAt"`
	ProcessedAt       *time.Time     `json:"processedAt,omitempty" yaml:"processedAt,omitempty"`
	LastError         string         `json:"lastError,omitempty" yaml:"lastError,omitempty"`
}

type OutEbicsRTNProvider struct {
	ID               int64      `json:"id" yaml:"id"`
	Name             string     `json:"name" yaml:"name"`
	Transport        string     `json:"transport" yaml:"transport"`
	Enabled          bool       `json:"enabled" yaml:"enabled"`
	SubscriberID     int64      `json:"subscriberId" yaml:"subscriberId"`
	AutoPullPolicy   string     `json:"autoPullPolicy" yaml:"autoPullPolicy"`
	LastConnectionAt *time.Time `json:"lastConnectionAt,omitempty" yaml:"lastConnectionAt,omitempty"`
	LastError        string     `json:"lastError,omitempty" yaml:"lastError,omitempty"`
}

type InEbicsRTNProvider struct {
	Name           string         `json:"name" yaml:"name"`
	Transport      string         `json:"transport" yaml:"transport"`
	Enabled        *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	SubscriberID   int64          `json:"subscriberId" yaml:"subscriberId"`
	Configuration  map[string]any `json:"configuration,omitempty" yaml:"configuration,omitempty"`
	AutoPullPolicy string         `json:"autoPullPolicy,omitempty" yaml:"autoPullPolicy,omitempty"`
}

type InEbicsRTNEventAction struct {
	Action   string         `json:"action" yaml:"action"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

## 8. Statuts minimaux a figer

## 8.1 `EbicsRTNEvent`

Statuts recommandes:

```go
const (
	RTNEventStatusReceived    = "RECEIVED"
	RTNEventStatusDuplicate   = "DUPLICATE"
	RTNEventStatusProcessing  = "PROCESSING"
	RTNEventStatusProcessed   = "PROCESSED"
	RTNEventStatusRetryable   = "RETRYABLE"
	RTNEventStatusQuarantined = "QUARANTINED"
	RTNEventStatusFailed      = "FAILED"
)
```

## 8.2 `EbicsRTNProvider`

Etats d'exploitation recommandes:

```go
const (
	RTNProviderStateStopped   = "STOPPED"
	RTNProviderStateStarting  = "STARTING"
	RTNProviderStateRunning   = "RUNNING"
	RTNProviderStateDegraded  = "DEGRADED"
	RTNProviderStateFailed    = "FAILED"
)
```

## 9. Point de vigilance principal

Le danger ici serait de faire porter trop de logique au provider `WSS`.

La bonne ligne reste:

- provider = transport et connexion;
- runtime = normalisation, idempotence, plan d'action;
- models = persistance et exploitation.
