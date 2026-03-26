# Phase B - Structs Go et signatures de methodes

## 1. Objet

Ce document fixe les declarations cibles de la `Phase B`.

Il couvre:

- les models `EbicsContractView`, `EbicsContractViewItem`,
  `EbicsPayloadProfile`;
- les structs runtime de resolution et de validation;
- les signatures de methodes et helpers associes.

## 2. Conventions

Regles a conserver:

- compatibilite multi-SGBD via XORM;
- types persistants simples et portables;
- validation dans `BeforeWrite(db database.Access) error`;
- pas de blob JSON structurel par defaut;
- pas de logique d'execution protocolaire dans cette phase.

Consequence:

- le modele logique SQL deja defini reste valable dans son intention;
- en revanche, la projection physique exacte en colonnes/types/contraintes doit
  rester sous le controle de l'ORM et des migrations compatibles multi-SGBD.

## 3. Constantes model

## 3.1 `pkg/model/table_names.go`

Ajouts cibles:

```go
const (
	TableEbicsContractViews     = "ebics_contract_views"
	TableEbicsContractViewItems = "ebics_contract_view_items"
	TableEbicsPayloadProfiles   = "ebics_payload_profiles"
)
```

## 3.2 `pkg/model/display_names.go`

Ajouts cibles:

```go
const (
	NameEbicsContractView     = "ebics contract view"
	NameEbicsContractViewItem = "ebics contract view item"
	NameEbicsPayloadProfile   = "ebics payload profile"
)
```

## 4. Models `pkg/model`

## 4.1 `pkg/model/ebics_contract_view.go`

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

type EbicsContractView struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID sql.NullInt64 `xorm:"ebics_subscriber_id"`

	SourceOrderType   string        `xorm:"source_order_type"`
	SourceOperationID sql.NullInt64 `xorm:"source_operation_id"`
	VersionTag        string        `xorm:"version_tag"`
	Status            string        `xorm:"status"`

	FetchedAt time.Time `xorm:"fetched_at DATETIME(6) UTC"`
	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsContractView) TableName() string
func (*EbicsContractView) Appellation() string
func (v *EbicsContractView) GetID() int64
func (v *EbicsContractView) BeforeWrite(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsContractViewStatus(status string) error
func validateEbicsContractSourceOrderType(orderType string) error
```

## 4.2 `pkg/model/ebics_contract_view_item.go`

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

type EbicsContractViewItem struct {
	ID             int64  `xorm:"<- id AUTOINCR"`
	Owner          string `xorm:"owner"`
	ContractViewID int64  `xorm:"contract_view_id"`

	ItemType           string `xorm:"item_type"`
	ItemKey            string `xorm:"item_key"`
	OrderType          string `xorm:"order_type"`
	ServiceName        string `xorm:"service_name"`
	ServiceOption      string `xorm:"service_option"`
	Scope              string `xorm:"scope"`
	MsgName            string `xorm:"msg_name"`
	ContainerType      string `xorm:"container_type"`
	AdminOrderType     string `xorm:"admin_order_type"`
	AuthorisationLevel string `xorm:"authorisation_level"`
	AccountID          string `xorm:"account_id"`
	MaxAmountValue     string `xorm:"max_amount_value"`
	MaxAmountCurrency  string `xorm:"max_amount_currency"`
	IsEnabled          bool   `xorm:"is_enabled"`
	Payload            string `xorm:"payload"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

Note de conception:

- pour rester prudent cote multi-SGBD, `Payload` est cadre ici comme `string`
  serialise plutot que `map[string]any`;
- il doit rester secondaire et non structurant.

### Methodes cibles

```go
func (*EbicsContractViewItem) TableName() string
func (*EbicsContractViewItem) Appellation() string
func (i *EbicsContractViewItem) GetID() int64
func (i *EbicsContractViewItem) BeforeWrite(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsContractItemType(itemType string) error
func validateEbicsContractItemCoherence(item *EbicsContractViewItem) error
```

## 4.3 `pkg/model/ebics_payload_profile.go`

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

type EbicsPayloadProfile struct {
	ID                     int64         `xorm:"<- id AUTOINCR"`
	Owner                  string        `xorm:"owner"`
	Name                   string        `xorm:"name"`
	Label                  string        `xorm:"label"`
	Description            string        `xorm:"description"`
	OrderType              string        `xorm:"order_type"`
	Direction              string        `xorm:"direction"`
	ServiceName            string        `xorm:"service_name"`
	ServiceOption          string        `xorm:"service_option"`
	Scope                  string        `xorm:"scope"`
	MsgName                string        `xorm:"msg_name"`
	ContainerType          string        `xorm:"container_type"`
	DefaultRuleID          sql.NullInt64 `xorm:"default_rule_id"`
	DefaultTargetDirectory string        `xorm:"default_target_directory"`
	RequiresDeclaredAmount bool          `xorm:"requires_declared_amount"`
	DefaultCurrency        string        `xorm:"default_currency"`
	AllowedExtensions      string        `xorm:"allowed_extensions"`
	FilenamePattern        string        `xorm:"filename_pattern"`
	StrictContractCheck    bool          `xorm:"strict_contract_check"`
	IsEnabled              bool          `xorm:"is_enabled"`
	Metadata               string        `xorm:"metadata"`
	CreatedAt              time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt              time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

Note de conception:

- `AllowedExtensions` et `Metadata` sont modeles ici comme `string` serialise
  pour rester prudents vis-a-vis de la portabilite ORM/SGBD;
- la restitution REST peut continuer a les exposer comme liste/map;
- la logique de serialisation doit rester centralisee.

### Methodes cibles

```go
func (*EbicsPayloadProfile) TableName() string
func (*EbicsPayloadProfile) Appellation() string
func (p *EbicsPayloadProfile) GetID() int64
func (p *EbicsPayloadProfile) BeforeWrite(db database.Access) error
func (p *EbicsPayloadProfile) AfterRead(database.ReadAccess) error
func (p *EbicsPayloadProfile) AfterInsert(db database.Access) error
func (p *EbicsPayloadProfile) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsPayloadOrderType(orderType string) error
func validateEbicsPayloadDirection(direction string) error
func validateEbicsPayloadProfileCoherence(p *EbicsPayloadProfile) error
func serializeStringSlice(values []string) (string, error)
func deserializeStringSlice(raw string) ([]string, error)
func serializeStringMap(raw map[string]any) (string, error)
func deserializeStringMap(raw string) (map[string]any, error)
```

## 5. Runtime `pkg/protocols/modules/ebics/runtime`

## 5.1 `payload_resolution.go`

### Structs cibles

```go
package runtime

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type PayloadRequestInput struct {
	ProfileName string
	RuleName    string
	Subscriber  PayloadSubscriberRef
	File        *PayloadFileRef
	Target      *PayloadTargetRef
	Service     *PayloadServiceRef
	Metadata    map[string]any
}

type PayloadSubscriberRef struct {
	HostID    string
	PartnerID string
	UserID    string
}

type PayloadFileRef struct {
	Path       string
	OutputName string
}

type PayloadTargetRef struct {
	Directory string
}

type PayloadServiceRef struct {
	ServiceName   string
	ServiceOption string
	Scope         string
	MsgName       string
	ContainerType string
}

type ResolvedPayloadRequest struct {
	OrderType         string
	ResolutionMode    string
	Profile           *model.EbicsPayloadProfile
	ProfileName       string
	RuleName          string
	Subscriber        PayloadSubscriberRef
	ResolvedFile      *PayloadFileRef
	ResolvedTarget    *PayloadTargetRef
	ResolvedService   PayloadServiceRef
	ResolvedMetadata  map[string]any
	DeclaredAmount    string
	DeclaredCurrency  string
	ContractViewID    int64
	ContractItemIDs   []int64
}
```

### Interface cible

```go
type PayloadProfileResolver interface {
	GetPayloadProfile(owner, name string) (*model.EbicsPayloadProfile, error)
}
```

### Signatures cibles

```go
func ResolvePayloadRequest(
	input PayloadRequestInput,
	profilePolicy string,
	defaults map[string]any,
	resolver PayloadProfileResolver,
) (*ResolvedPayloadRequest, error)
```

### Helpers prives recommandes

```go
func resolvePayloadService(input PayloadRequestInput, profile *model.EbicsPayloadProfile, defaults map[string]any) (PayloadServiceRef, error)
func resolvePayloadTarget(input PayloadRequestInput, profile *model.EbicsPayloadProfile) *PayloadTargetRef
func resolvePayloadFile(input PayloadRequestInput) *PayloadFileRef
func mergePayloadMetadata(input map[string]any, defaults map[string]any) map[string]any
func extractDeclaredAmount(metadata map[string]any) (amount string, currency string)
```

## 5.2 `contract_validation.go`

### Structs cibles

```go
package runtime

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type ContractValidationResult struct {
	Status         string
	Message        string
	ContractViewID int64
	MatchedItems   []model.EbicsContractViewItem
}
```

### Interface cible

```go
type ContractViewResolver interface {
	GetActiveContractView(
		owner string,
		hostID string,
		partnerID string,
		userID string,
	) (*model.EbicsContractView, []model.EbicsContractViewItem, error)
}
```

### Signatures cibles

```go
func ValidateResolvedPayloadRequest(
	owner string,
	request *ResolvedPayloadRequest,
	resolver ContractViewResolver,
) (*ContractValidationResult, error)
```

### Helpers prives recommandes

```go
func matchContractItems(
	request *ResolvedPayloadRequest,
	items []model.EbicsContractViewItem,
) []model.EbicsContractViewItem

func isContractItemCompatible(
	request *ResolvedPayloadRequest,
	item model.EbicsContractViewItem,
) bool
```

## 6. DTO API `pkg/admin/rest/api`

## 6.1 `ebics_payload_profiles.go`

### Structs cibles

```go
type InEbicsPayloadProfile struct {
	Name                   string         `json:"name" yaml:"name"`
	Label                  string         `json:"label,omitempty" yaml:"label,omitempty"`
	Description            string         `json:"description,omitempty" yaml:"description,omitempty"`
	OrderType              string         `json:"orderType" yaml:"orderType"`
	Direction              string         `json:"direction" yaml:"direction"`
	ServiceName            string         `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption          string         `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope                  string         `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName                string         `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType          string         `json:"containerType,omitempty" yaml:"containerType,omitempty"`
	DefaultRule            string         `json:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	DefaultTargetDirectory string         `json:"defaultTargetDirectory,omitempty" yaml:"defaultTargetDirectory,omitempty"`
	RequiresDeclaredAmount bool           `json:"requiresDeclaredAmount,omitempty" yaml:"requiresDeclaredAmount,omitempty"`
	DefaultCurrency        string         `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
	AllowedExtensions      []string       `json:"allowedExtensions,omitempty" yaml:"allowedExtensions,omitempty"`
	FilenamePattern        string         `json:"filenamePattern,omitempty" yaml:"filenamePattern,omitempty"`
	StrictContractCheck    *bool          `json:"strictContractCheck,omitempty" yaml:"strictContractCheck,omitempty"`
	IsEnabled              *bool          `json:"isEnabled,omitempty" yaml:"isEnabled,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type OutEbicsPayloadProfile struct {
	ID                     int64          `json:"id" yaml:"id"`
	Name                   string         `json:"name" yaml:"name"`
	Label                  string         `json:"label,omitempty" yaml:"label,omitempty"`
	Description            string         `json:"description,omitempty" yaml:"description,omitempty"`
	OrderType              string         `json:"orderType" yaml:"orderType"`
	Direction              string         `json:"direction" yaml:"direction"`
	ServiceName            string         `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption          string         `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope                  string         `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName                string         `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType          string         `json:"containerType,omitempty" yaml:"containerType,omitempty"`
	DefaultRule            string         `json:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	DefaultTargetDirectory string         `json:"defaultTargetDirectory,omitempty" yaml:"defaultTargetDirectory,omitempty"`
	RequiresDeclaredAmount bool           `json:"requiresDeclaredAmount" yaml:"requiresDeclaredAmount"`
	DefaultCurrency        string         `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
	AllowedExtensions      []string       `json:"allowedExtensions,omitempty" yaml:"allowedExtensions,omitempty"`
	FilenamePattern        string         `json:"filenamePattern,omitempty" yaml:"filenamePattern,omitempty"`
	StrictContractCheck    bool           `json:"strictContractCheck" yaml:"strictContractCheck"`
	IsEnabled              bool           `json:"isEnabled" yaml:"isEnabled"`
	Metadata               map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

## 6.2 `ebics_contract_views.go`

### Structs cibles

```go
type OutEbicsContractView struct {
	ID               int64     `json:"id" yaml:"id"`
	HostID           string    `json:"hostId" yaml:"hostId"`
	PartnerID        string    `json:"partnerId,omitempty" yaml:"partnerId,omitempty"`
	UserID           string    `json:"userId,omitempty" yaml:"userId,omitempty"`
	SourceOrderType  string    `json:"sourceOrderType" yaml:"sourceOrderType"`
	VersionTag       string    `json:"versionTag,omitempty" yaml:"versionTag,omitempty"`
	Status           string    `json:"status" yaml:"status"`
	FetchedAt        time.Time `json:"fetchedAt" yaml:"fetchedAt"`
}

type OutEbicsContractViewItem struct {
	ID                 int64  `json:"id" yaml:"id"`
	ItemType           string `json:"itemType" yaml:"itemType"`
	ItemKey            string `json:"itemKey" yaml:"itemKey"`
	OrderType          string `json:"orderType,omitempty" yaml:"orderType,omitempty"`
	ServiceName        string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption      string `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope              string `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName            string `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType      string `json:"containerType,omitempty" yaml:"containerType,omitempty"`
	AdminOrderType     string `json:"adminOrderType,omitempty" yaml:"adminOrderType,omitempty"`
	AuthorisationLevel string `json:"authorisationLevel,omitempty" yaml:"authorisationLevel,omitempty"`
	AccountID          string `json:"accountId,omitempty" yaml:"accountId,omitempty"`
	MaxAmountValue     string `json:"maxAmountValue,omitempty" yaml:"maxAmountValue,omitempty"`
	MaxAmountCurrency  string `json:"maxAmountCurrency,omitempty" yaml:"maxAmountCurrency,omitempty"`
	IsEnabled          bool   `json:"isEnabled" yaml:"isEnabled"`
}
```

## 7. Point de vigilance principal

La `Phase B` ne doit pas rebasculer vers une architecture trop “souple” qui
deviendrait en pratique floue a exploiter.

Donc:

- colonnes structurelles d'abord;
- serialisation libre seulement en second rang;
- validation explicable;
- et distinction nette entre profil, contrat et demande resolue.
