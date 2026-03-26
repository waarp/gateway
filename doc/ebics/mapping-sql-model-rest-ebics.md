# Mapping cible - SQL -> structs model -> DTO REST EBICS

## 1. Objet

Ce document derive le modele SQL detaille vers:

- les structs `model` cibles;
- les DTO REST cibles;
- les relations minimales entre persistance et administration.

Il couvre en priorite:

- `ebics_operations`
- `ebics_transactions`
- `ebics_transaction_segments`

## 2. Principes

- un objet SQL durable correspond a un struct `model` explicite;
- les DTO REST n'exposent pas tous les champs internes;
- les champs purement techniques de persistence restent dans `metadata` quand
  ils ne meritent pas un champ REST dedie;
- les relations avec `Transfer` sont visibles en REST mais restent optionnelles.
- pour rester coherents avec la contrainte multi-SGBD, les champs libres
  persistes sont de preference stockes sous forme `string` serialisee cote
  `model`, puis exposes comme objets/listes cote DTO REST.

## 3. `ebics_operations`

## 3.1 Table SQL

Table source:

- [sql-modele-detaille-ebics.md](/c:/MonProjet/Waarp-Gateway/doc/ebics/sql-modele-detaille-ebics.md)

## 3.2 Struct `model.EbicsOperation`

Struct cible recommandee:

```go
type EbicsOperation struct {
    ID                int64          `xorm:"<- id AUTOINCR"`
    Owner             string         `xorm:"owner"`
    LocalAgentID      sql.NullInt64  `xorm:"local_agent_id"`
    ClientID          sql.NullInt64  `xorm:"client_id"`
    RemoteAgentID     sql.NullInt64  `xorm:"remote_agent_id"`
    LocalAccountID    sql.NullInt64  `xorm:"local_account_id"`
    RemoteAccountID   sql.NullInt64  `xorm:"remote_account_id"`
    EbicsHostID       int64          `xorm:"ebics_host_id"`
    EbicsSubscriberID int64          `xorm:"ebics_subscriber_id"`
    OperationType     string         `xorm:"operation_type"`
    OrderType         string         `xorm:"order_type"`
    Direction         string         `xorm:"direction"`
    TransportMode     string         `xorm:"transport_mode"`
    TransactionID     string         `xorm:"transaction_id"`
    RequestID         string         `xorm:"request_id"`
    CorrelationID     string         `xorm:"correlation_id"`
    EbicsVersion      string         `xorm:"ebics_version"`
    Status            string         `xorm:"status"`
    Severity          string         `xorm:"severity"`
    TechnicalReturnCode    string         `xorm:"technical_return_code"`
    TechnicalReturnMessage string         `xorm:"technical_return_message"`
    BusinessReturnCode     string         `xorm:"business_return_code"`
    BusinessReturnMessage  string         `xorm:"business_return_message"`
    GatewayOutcome         string         `xorm:"gateway_outcome"`
    RetryDecision          string         `xorm:"retry_decision"`
    ManualActionRequired   bool           `xorm:"manual_action_required"`
    TransferID        sql.NullInt64  `xorm:"transfer_id"`
    ContractViewID    sql.NullInt64  `xorm:"contract_view_id"`
    RTNEventID        sql.NullInt64  `xorm:"rtn_event_id"`
    Metadata          string         `xorm:"metadata"`
    StartedAt         time.Time      `xorm:"started_at DATETIME(6) UTC"`
    FinishedAt        time.Time      `xorm:"finished_at DATETIME(6) UTC"`
    CreatedAt         time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt         time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

Validations minimales:

- `operation_type` obligatoire;
- `order_type` obligatoire;
- `status` obligatoire;
- `direction` obligatoire;
- `severity` obligatoire;
- coherence minimale de rattachement client/server;
- `transfer_id` interdit si l'ordre ne doit jamais projeter un transfert.

## 3.3 DTO REST

`api.OutEbicsOperation`

```go
type OutEbicsOperation struct {
    ID             int64          `json:"id" yaml:"id"`
    OperationType  string         `json:"operationType" yaml:"operationType"`
    OrderType      string         `json:"orderType" yaml:"orderType"`
    Direction      string         `json:"direction" yaml:"direction"`
    TransportMode  string         `json:"transportMode" yaml:"transportMode"`
    Status         string         `json:"status" yaml:"status"`
    Severity       string         `json:"severity" yaml:"severity"`
    HostID         string         `json:"hostId" yaml:"hostId"`
    PartnerID      string         `json:"partnerId" yaml:"partnerId"`
    UserID         string         `json:"userId" yaml:"userId"`
    TransactionID  string         `json:"transactionId,omitempty" yaml:"transactionId,omitempty"`
    RequestID      string         `json:"requestId,omitempty" yaml:"requestId,omitempty"`
    CorrelationID  string         `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
    Technical      map[string]string `json:"technical,omitempty" yaml:"technical,omitempty"`
    Business       map[string]string `json:"business,omitempty" yaml:"business,omitempty"`
    GatewayOutcome string            `json:"gatewayOutcome,omitempty" yaml:"gatewayOutcome,omitempty"`
    RetryDecision  string            `json:"retryDecision,omitempty" yaml:"retryDecision,omitempty"`
    ManualActionRequired bool        `json:"manualActionRequired" yaml:"manualActionRequired"`
    TransferID     Nullable[int64] `json:"transferId,omitzero" yaml:"transferId,omitempty"`
    StartedAt      Nullable[time.Time] `json:"startedAt,omitzero" yaml:"startedAt,omitempty"`
    FinishedAt     Nullable[time.Time] `json:"finishedAt,omitzero" yaml:"finishedAt,omitempty"`
    Metadata       map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

`api.InEbicsOperationAction`

```go
type InEbicsOperationAction struct {
    Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

## 4. `ebics_transactions`

## 4.1 Struct `model.EbicsTransaction`

```go
type EbicsTransaction struct {
    ID                int64          `xorm:"<- id AUTOINCR"`
    Owner             string         `xorm:"owner"`
    EbicsOperationID  sql.NullInt64  `xorm:"ebics_operation_id"`
    EbicsHostID       int64          `xorm:"ebics_host_id"`
    EbicsSubscriberID int64          `xorm:"ebics_subscriber_id"`
    TransactionID     string         `xorm:"transaction_id"`
    OrderType         string         `xorm:"order_type"`
    TransferID        sql.NullInt64  `xorm:"transfer_id"`
    Status            string         `xorm:"status"`
    Direction         string         `xorm:"direction"`
    SegmentCount      int            `xorm:"segment_count"`
    CurrentSegment    int            `xorm:"current_segment"`
    TotalSize         int64          `xorm:"total_size"`
    ResumedFromTxID   sql.NullInt64  `xorm:"resumed_from_tx_id"`
    Metadata          string         `xorm:"metadata"`
    CreatedAt         time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt         time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

## 4.2 DTO REST

REST de phase 1:

- exposition surtout en lecture depuis le detail d'operation;
- pas necessairement une famille de routes publique autonome en premier.

DTO de detail recommande:

```go
type OutEbicsTransaction struct {
    ID             int64  `json:"id" yaml:"id"`
    TransactionID  string `json:"transactionId" yaml:"transactionId"`
    OrderType      string `json:"orderType" yaml:"orderType"`
    Status         string `json:"status" yaml:"status"`
    Direction      string `json:"direction" yaml:"direction"`
    SegmentCount   int    `json:"segmentCount" yaml:"segmentCount"`
    CurrentSegment int    `json:"currentSegment" yaml:"currentSegment"`
    TotalSize      int64  `json:"totalSize" yaml:"totalSize"`
    TransferID     Nullable[int64] `json:"transferId,omitzero" yaml:"transferId,omitempty"`
}
```

## 5. `ebics_transaction_segments`

## 5.1 Struct `model.EbicsTransactionSegment`

```go
type EbicsTransactionSegment struct {
    ID                 int64          `xorm:"<- id AUTOINCR"`
    Owner              string         `xorm:"owner"`
    EbicsTransactionID int64          `xorm:"ebics_transaction_id"`
    SegmentNumber      int            `xorm:"segment_number"`
    SegmentStatus      string         `xorm:"segment_status"`
    PayloadSize        int64          `xorm:"payload_size"`
    Checksum           string         `xorm:"checksum"`
    StoredPayloadRef   string         `xorm:"stored_payload_ref"`
    Metadata           string         `xorm:"metadata"`
    CreatedAt          time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt          time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

## 5.2 DTO REST

REST de phase 1:

- lecture dans le detail de transaction ou d'operation;
- pas d'action directe exposee aux exploitants en premiere intention.

## 6. `ebics_contract_views`

## 6.1 Struct `model.EbicsContractView`

```go
type EbicsContractView struct {
    ID                 int64          `xorm:"<- id AUTOINCR"`
    Owner              string         `xorm:"owner"`
    EbicsHostID        int64          `xorm:"ebics_host_id"`
    EbicsSubscriberID  sql.NullInt64  `xorm:"ebics_subscriber_id"`
    SourceOrderType    string         `xorm:"source_order_type"`
    SourceOperationID  sql.NullInt64  `xorm:"source_operation_id"`
    VersionTag         string         `xorm:"version_tag"`
    Status             string         `xorm:"status"`
    FetchedAt          time.Time      `xorm:"fetched_at DATETIME(6) UTC"`
    CreatedAt          time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt          time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

## 6.2 DTO REST

DTO cible:

- `OutEbicsContractView`
- `OutEbicsContractViewItem`

Le detail fin ne devrait pas rester majoritairement en JSON libre.

## 6.3 Struct `model.EbicsContractViewItem`

```go
type EbicsContractViewItem struct {
    ID                 int64          `xorm:"<- id AUTOINCR"`
    Owner              string         `xorm:"owner"`
    ContractViewID     int64          `xorm:"contract_view_id"`
    ItemType           string         `xorm:"item_type"`
    ItemKey            string         `xorm:"item_key"`
    OrderType          string         `xorm:"order_type"`
    ServiceName        string         `xorm:"service_name"`
    ServiceOption      string         `xorm:"service_option"`
    Scope              string         `xorm:"scope"`
    MsgName            string         `xorm:"msg_name"`
    ContainerType      string         `xorm:"container_type"`
    AdminOrderType     string         `xorm:"admin_order_type"`
    AuthorisationLevel string         `xorm:"authorisation_level"`
    AccountID          string         `xorm:"account_id"`
    MaxAmountValue     string         `xorm:"max_amount_value"`
    MaxAmountCurrency  string         `xorm:"max_amount_currency"`
    IsEnabled          bool           `xorm:"is_enabled"`
    Payload            string         `xorm:"payload"`
    CreatedAt          time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt          time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

## 7. `ebics_rtn_events`

## 7.1 Struct `model.EbicsRTNEvent`

```go
type EbicsRTNEvent struct {
    ID                int64          `xorm:"<- id AUTOINCR"`
    Owner             string         `xorm:"owner"`
    Source            string         `xorm:"source"`
    EventID           string         `xorm:"event_id"`
    CorrelationID     string         `xorm:"correlation_id"`
    IdempotenceKey    string         `xorm:"idempotence_key"`
    EbicsHostID       sql.NullInt64  `xorm:"ebics_host_id"`
    EbicsSubscriberID sql.NullInt64  `xorm:"ebics_subscriber_id"`
    OrderTypeHint     string         `xorm:"order_type_hint"`
    ProfileID         string         `xorm:"profile_id"`
    Payload           string         `xorm:"payload"`
    Status            string         `xorm:"status"`
    Attempts          int            `xorm:"attempts"`
    NextRetryAt       time.Time      `xorm:"next_retry_at DATETIME(6) UTC"`
    ReceivedAt        time.Time      `xorm:"received_at DATETIME(6) UTC"`
    ProcessedAt       time.Time      `xorm:"processed_at DATETIME(6) UTC"`
    CreatedAt         time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt         time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

## 7.2 DTO REST

DTO cible:

- `OutEbicsRTNEvent`
- `InEbicsRTNProvider`
- `OutEbicsRTNProvider`

Important:

- le detail du transport RTN doit rester separe de la charge utile normalisee;
- un evenement RTN peut etre correle a une ou plusieurs `EbicsOperation`.

## 8. `ebics_payload_profiles`

Struct cible:

- `model.EbicsPayloadProfile`

DTO cibles:

- `OutEbicsPayloadProfile`
- `InEbicsPayloadProfile`
- `InEbicsPayloadProfileUpdate`

Principes:

- `DefaultRuleID` est resolu en nom de `Rule` en REST quand c'est possible;
- le profil payload ne duplique pas une seconde vue contractuelle;
- l'objet doit rester lisible pour l'exploitation.

## 9. Choix RTN de phase 1

Pour la premiere implementation RTN dans Gateway, la cible recommandee est:

- une couche de transport `WebSocket / WSS` comme canal primaire d'ingestion.

Cette decision est coherente avec la documentation de `lib-ebics`, qui presente
RTN comme un canal hors bande typiquement `WebSocket/WSS`, tout en gardant le
coeur RTN agnostique du transport.

Consequence de design:

- les structs `model` et DTO REST ne doivent pas etre couples au WebSocket;
- le WebSocket est un provider/adapter de transport;
- la persistance RTN stocke des evenements normalises.

## 10. Families REST recommandees

Phase 1:

- `/api/ebics/operations`
- `/api/ebics/payload-profiles`
- `/api/ebics/partners/{partner}/contract-view`
- `/api/ebics/rtn/events`

Phase 2 si necessaire:

- `/api/ebics/transactions`
- `/api/ebics/transactions/{id}/segments`

## 11. Resultat attendu

Si cette derivation est respectee:

- les models Go seront presque mecaniques a ecrire;
- les DTO REST resteront lisibles;
- la separation `operation / transaction / segment / transfer` restera stable;
- l'implementation RTN par WebSocket restera un detail de provider, pas une
  pollution du coeur EBICS.
