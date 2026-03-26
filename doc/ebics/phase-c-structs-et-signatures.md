# Phase C - Structs Go et signatures de methodes

## 1. Objet

Ce document fixe les declarations cibles de la `Phase C`.

Il couvre:

- `EbicsOperation`;
- `EbicsTransaction`;
- `EbicsTransactionSegment`;
- les structs runtime de mapping d'operation et de policy retry;
- les DTO REST minimaux pour payloads, operations et transactions.

## 2. Conventions

Regles a conserver:

- compatibilite multi-SGBD via XORM;
- separation stricte `operation / transaction / segment / transfer`;
- return codes `technical` et `business` toujours separes;
- `Transfer` reste optionnel et derive;
- pas de logique metier bancaire embarquee.

## 3. Constantes model

## 3.1 `pkg/model/table_names.go`

Ajouts cibles:

```go
const (
	TableEbicsOperations          = "ebics_operations"
	TableEbicsTransactions        = "ebics_transactions"
	TableEbicsTransactionSegments = "ebics_transaction_segments"
)
```

## 3.2 `pkg/model/display_names.go`

Ajouts cibles:

```go
const (
	NameEbicsOperation          = "ebics operation"
	NameEbicsTransaction        = "ebics transaction"
	NameEbicsTransactionSegment = "ebics transaction segment"
)
```

## 4. Models `pkg/model`

## 4.1 `pkg/model/ebics_operation.go`

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

type EbicsOperation struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	LocalAgentID      sql.NullInt64 `xorm:"local_agent_id"`
	ClientID          sql.NullInt64 `xorm:"client_id"`
	RemoteAgentID     sql.NullInt64 `xorm:"remote_agent_id"`
	LocalAccountID    sql.NullInt64 `xorm:"local_account_id"`
	RemoteAccountID   sql.NullInt64 `xorm:"remote_account_id"`
	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`

	OperationType string `xorm:"operation_type"`
	OrderType     string `xorm:"order_type"`
	Direction     string `xorm:"direction"`
	TransportMode string `xorm:"transport_mode"`

	TransactionID string `xorm:"transaction_id"`
	RequestID     string `xorm:"request_id"`
	CorrelationID string `xorm:"correlation_id"`
	EbicsVersion  string `xorm:"ebics_version"`

	Status                  string `xorm:"status"`
	Severity                string `xorm:"severity"`
	TechnicalReturnCode     string `xorm:"technical_return_code"`
	TechnicalReturnMessage  string `xorm:"technical_return_message"`
	BusinessReturnCode      string `xorm:"business_return_code"`
	BusinessReturnMessage   string `xorm:"business_return_message"`
	GatewayOutcome          string `xorm:"gateway_outcome"`
	RetryDecision           string `xorm:"retry_decision"`
	ManualActionRequired    bool   `xorm:"manual_action_required"`

	TransferID     sql.NullInt64 `xorm:"transfer_id"`
	ContractViewID sql.NullInt64 `xorm:"contract_view_id"`
	RTNEventID     sql.NullInt64 `xorm:"rtn_event_id"`
	Metadata       string        `xorm:"metadata"`

	StartedAt  time.Time `xorm:"started_at DATETIME(6) UTC"`
	FinishedAt time.Time `xorm:"finished_at DATETIME(6) UTC"`
	CreatedAt  time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt  time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsOperation) TableName() string
func (*EbicsOperation) Appellation() string
func (o *EbicsOperation) GetID() int64
func (o *EbicsOperation) BeforeWrite(db database.Access) error
func (o *EbicsOperation) AfterRead(database.ReadAccess) error
func (o *EbicsOperation) AfterInsert(db database.Access) error
func (o *EbicsOperation) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsOperationType(value string) error
func validateEbicsOperationStatus(value string) error
func validateEbicsOperationSeverity(value string) error
func validateEbicsOperationDirection(value string) error
func validateEbicsTransportMode(value string) error
func validateEbicsGatewayOutcome(value string) error
func validateEbicsRetryDecision(value string) error
func validateEbicsOperationBinding(o *EbicsOperation) error
```

## 4.2 `pkg/model/ebics_transaction.go`

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

type EbicsTransaction struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	EbicsOperationID  sql.NullInt64 `xorm:"ebics_operation_id"`
	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`

	TransactionID   string        `xorm:"transaction_id"`
	OrderType       string        `xorm:"order_type"`
	TransferID      sql.NullInt64 `xorm:"transfer_id"`
	Status          string        `xorm:"status"`
	Direction       string        `xorm:"direction"`
	SegmentCount    int           `xorm:"segment_count"`
	CurrentSegment  int           `xorm:"current_segment"`
	TotalSize       int64         `xorm:"total_size"`
	ResumedFromTxID sql.NullInt64 `xorm:"resumed_from_tx_id"`
	Metadata        string        `xorm:"metadata"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsTransaction) TableName() string
func (*EbicsTransaction) Appellation() string
func (t *EbicsTransaction) GetID() int64
func (t *EbicsTransaction) BeforeWrite(db database.Access) error
func (t *EbicsTransaction) AfterRead(database.ReadAccess) error
func (t *EbicsTransaction) AfterInsert(db database.Access) error
func (t *EbicsTransaction) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsTransactionStatus(value string) error
func validateEbicsTransactionDirection(value string) error
func validateEbicsTransactionCounters(t *EbicsTransaction) error
```

## 4.3 `pkg/model/ebics_transaction_segment.go`

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

type EbicsTransactionSegment struct {
	ID                 int64  `xorm:"<- id AUTOINCR"`
	Owner              string `xorm:"owner"`
	EbicsTransactionID int64  `xorm:"ebics_transaction_id"`

	SegmentNumber    int    `xorm:"segment_number"`
	SegmentStatus    string `xorm:"segment_status"`
	PayloadSize      int64  `xorm:"payload_size"`
	Checksum         string `xorm:"checksum"`
	StoredPayloadRef string `xorm:"stored_payload_ref"`
	Metadata         string `xorm:"metadata"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsTransactionSegment) TableName() string
func (*EbicsTransactionSegment) Appellation() string
func (s *EbicsTransactionSegment) GetID() int64
func (s *EbicsTransactionSegment) BeforeWrite(db database.Access) error
func (s *EbicsTransactionSegment) AfterRead(database.ReadAccess) error
func (s *EbicsTransactionSegment) AfterInsert(db database.Access) error
func (s *EbicsTransactionSegment) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsSegmentStatus(value string) error
```

## 5. Runtime `pkg/protocols/modules/ebics/runtime`

## 5.1 `operation_mapper.go`

### Structs cibles

```go
package runtime

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type OperationMappingInput struct {
	Owner              string
	LocalAgentID       int64
	ClientID           int64
	RemoteAgentID      int64
	LocalAccountID     int64
	RemoteAccountID    int64
	EbicsHostID        int64
	EbicsSubscriberID  int64
	OrderType          string
	OperationType      string
	Direction          string
	TransportMode      string
	CorrelationID      string
	ContractViewID     int64
	ResolvedRequest    *ResolvedPayloadRequest
}
```

### Signatures cibles

```go
func NewPayloadOperation(input OperationMappingInput) (*model.EbicsOperation, error)
func BindTransferToOperation(op *model.EbicsOperation, transferID int64) error
func UpdateOperationOutcomeFromReturnCodes(op *model.EbicsOperation, technicalCode, technicalMsg, businessCode, businessMsg string) error
```

## 5.2 `retry_policy.go`

### Struct cible

```go
package runtime

type RetryPolicyDecision struct {
	GatewayOutcome       string
	RetryDecision        string
	ManualActionRequired bool
	Message              string
}
```

### Signatures cibles

```go
func DecideRetryPolicy(
	orderType string,
	technicalCode string,
	businessCode string,
) (*RetryPolicyDecision, error)
```

### Helpers prives recommandes

```go
func isManualReplayOrder(orderType string) bool
func isPayloadOrder(orderType string) bool
```

## 6. Stores `pkg/protocols/modules/ebics/stores`

## 6.1 `operation_store.go`

### Interface cible

```go
package stores

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type OperationStore interface {
	InsertOperation(op *model.EbicsOperation) error
	UpdateOperation(op *model.EbicsOperation) error
	GetOperationByID(id int64) (*model.EbicsOperation, error)
	GetOperationByCorrelationID(owner, correlationID string) (*model.EbicsOperation, error)
}
```

## 6.2 `tx_store.go`

### Interfaces cibles

```go
package stores

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type TransactionStore interface {
	InsertTransaction(tx *model.EbicsTransaction) error
	UpdateTransaction(tx *model.EbicsTransaction) error
	GetTransactionByTxID(owner, txID string) (*model.EbicsTransaction, error)
}

type SegmentStore interface {
	InsertSegment(seg *model.EbicsTransactionSegment) error
	UpdateSegment(seg *model.EbicsTransactionSegment) error
	ListSegments(transactionID int64) ([]model.EbicsTransactionSegment, error)
}
```

## 7. DTO API `pkg/admin/rest/api`

## 7.1 `ebics_payload_requests.go`

### Structs cibles

```go
type InEbicsPayloadRequest struct {
	Profile    string          `json:"profile,omitempty" yaml:"profile,omitempty"`
	Rule       string          `json:"rule,omitempty" yaml:"rule,omitempty"`
	Subscriber InSubscriberRef `json:"subscriber" yaml:"subscriber"`
	File       *InPayloadFile  `json:"file,omitempty" yaml:"file,omitempty"`
	Target     *InPayloadTarget `json:"target,omitempty" yaml:"target,omitempty"`
	Service    *InPayloadService `json:"service,omitempty" yaml:"service,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type OutEbicsPayloadSubmission struct {
	OperationID         int64    `json:"operationId" yaml:"operationId"`
	OrderType           string   `json:"orderType" yaml:"orderType"`
	Status              string   `json:"status" yaml:"status"`
	CorrelationID       string   `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	TransferID          *int64   `json:"transferId,omitempty" yaml:"transferId,omitempty"`
	ContractViewID      *int64   `json:"contractViewId,omitempty" yaml:"contractViewId,omitempty"`
	MatchedContractItemIDs []int64 `json:"matchedContractItemIds,omitempty" yaml:"matchedContractItemIds,omitempty"`
}
```

## 7.2 `ebics_operations.go`

### Struct cible

```go
type OutEbicsOperation struct {
	ID                   int64          `json:"id" yaml:"id"`
	OperationType        string         `json:"operationType" yaml:"operationType"`
	OrderType            string         `json:"orderType" yaml:"orderType"`
	Direction            string         `json:"direction" yaml:"direction"`
	TransportMode        string         `json:"transportMode" yaml:"transportMode"`
	Status               string         `json:"status" yaml:"status"`
	Severity             string         `json:"severity" yaml:"severity"`
	TransactionID        string         `json:"transactionId,omitempty" yaml:"transactionId,omitempty"`
	RequestID            string         `json:"requestId,omitempty" yaml:"requestId,omitempty"`
	CorrelationID        string         `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	TechnicalReturnCode  string         `json:"technicalReturnCode,omitempty" yaml:"technicalReturnCode,omitempty"`
	TechnicalReturnMessage string       `json:"technicalReturnMessage,omitempty" yaml:"technicalReturnMessage,omitempty"`
	BusinessReturnCode   string         `json:"businessReturnCode,omitempty" yaml:"businessReturnCode,omitempty"`
	BusinessReturnMessage string        `json:"businessReturnMessage,omitempty" yaml:"businessReturnMessage,omitempty"`
	GatewayOutcome       string         `json:"gatewayOutcome,omitempty" yaml:"gatewayOutcome,omitempty"`
	RetryDecision        string         `json:"retryDecision,omitempty" yaml:"retryDecision,omitempty"`
	ManualActionRequired bool           `json:"manualActionRequired" yaml:"manualActionRequired"`
	TransferID           *int64         `json:"transferId,omitempty" yaml:"transferId,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

## 7.3 `ebics_transactions.go`

### Structs cibles

```go
type OutEbicsTransaction struct {
	ID             int64   `json:"id" yaml:"id"`
	TransactionID  string  `json:"transactionId" yaml:"transactionId"`
	OrderType      string  `json:"orderType" yaml:"orderType"`
	Status         string  `json:"status" yaml:"status"`
	Direction      string  `json:"direction" yaml:"direction"`
	SegmentCount   int     `json:"segmentCount" yaml:"segmentCount"`
	CurrentSegment int     `json:"currentSegment" yaml:"currentSegment"`
	TotalSize      int64   `json:"totalSize" yaml:"totalSize"`
	TransferID     *int64  `json:"transferId,omitempty" yaml:"transferId,omitempty"`
}

type OutEbicsTransactionSegment struct {
	ID               int64  `json:"id" yaml:"id"`
	SegmentNumber    int    `json:"segmentNumber" yaml:"segmentNumber"`
	SegmentStatus    string `json:"segmentStatus" yaml:"segmentStatus"`
	PayloadSize      int64  `json:"payloadSize" yaml:"payloadSize"`
	Checksum         string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	StoredPayloadRef string `json:"storedPayloadRef,omitempty" yaml:"storedPayloadRef,omitempty"`
}
```

## 8. Point de vigilance principal

La `Phase C` doit rester lisible pour l'exploitation.

Donc:

- un objet par responsabilite principale;
- aucune fusion artificielle des identifiants;
- aucune reduction des deux scopes EBICS a un seul statut;
- et aucune confusion entre reprise protocolaire et reprise fichier Gateway.
