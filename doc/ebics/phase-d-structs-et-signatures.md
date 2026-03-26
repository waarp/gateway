# Phase D - Structs Go et signatures de methodes

## 1. Objet

Ce document fixe les declarations cibles de la `Phase D`.

Il couvre:

- `EbicsKeyLifecycle`;
- `EbicsInitializationWorkflow`;
- le champ logique `signatureState`;
- les structs runtime et signatures des runners associes.

## 2. Conventions

Regles a conserver:

- `Credential` reste le conteneur cryptographique generique;
- les workflows sensibles sont portes par des objets EBICS dedies;
- toute transition manuelle importante doit etre traçable;
- compatibilite multi-SGBD via XORM;
- pas de logique metier de decision humaine dans Gateway.

## 3. Constantes model

## 3.1 `pkg/model/table_names.go`

Ajouts cibles:

```go
const (
	TableEbicsKeyLifecycles           = "ebics_key_lifecycles"
	TableEbicsInitializationWorkflows = "ebics_initialization_workflows"
)
```

## 3.2 `pkg/model/display_names.go`

Ajouts cibles:

```go
const (
	NameEbicsKeyLifecycle           = "ebics key lifecycle"
	NameEbicsInitializationWorkflow = "ebics initialization workflow"
)
```

## 4. Models `pkg/model`

## 4.1 `pkg/model/ebics_key_lifecycle.go`

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

type EbicsKeyLifecycle struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`

	KeyUsage             string        `xorm:"key_usage"`
	RotationType         string        `xorm:"rotation_type"`
	Status               string        `xorm:"status"`
	CurrentCredentialID  int64         `xorm:"current_credential_id"`
	NextCredentialID     sql.NullInt64 `xorm:"next_credential_id"`
	TriggerOperationID   sql.NullInt64 `xorm:"trigger_operation_id"`
	LastOperationID      sql.NullInt64 `xorm:"last_operation_id"`
	RequestedAt          time.Time     `xorm:"requested_at DATETIME(6) UTC"`
	SentAt               time.Time     `xorm:"sent_at DATETIME(6) UTC"`
	ActivatedAt          time.Time     `xorm:"activated_at DATETIME(6) UTC"`
	RetiredAt            time.Time     `xorm:"retired_at DATETIME(6) UTC"`
	Operator             string        `xorm:"operator"`
	Reason               string        `xorm:"reason"`
	Evidence             string        `xorm:"evidence"`
	CreatedAt            time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt            time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsKeyLifecycle) TableName() string
func (*EbicsKeyLifecycle) Appellation() string
func (l *EbicsKeyLifecycle) GetID() int64
func (l *EbicsKeyLifecycle) BeforeWrite(db database.Access) error
func (l *EbicsKeyLifecycle) AfterRead(database.ReadAccess) error
func (l *EbicsKeyLifecycle) AfterInsert(db database.Access) error
func (l *EbicsKeyLifecycle) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsKeyUsage(value string) error
func validateEbicsRotationType(value string) error
func validateEbicsKeyLifecycleStatus(value string) error
func validateEbicsKeyLifecycleBinding(l *EbicsKeyLifecycle) error
```

### Regles de validation attendues

- `EbicsSubscriberID` obligatoire;
- `KeyUsage` borne;
- `RotationType` borne;
- `Status` borne;
- `CurrentCredentialID` obligatoire;
- `NextCredentialID` obligatoire a partir de `MATERIAL_PREPARED`;
- impossibilite de referencer deux fois le meme credential comme `current` et
  `next`;
- verification d'existence des credentials references;
- unicite logique du lifecycle actif par `(subscriber, key_usage)`.

## 4.2 `pkg/model/ebics_initialization_workflow.go`

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

type EbicsInitializationWorkflow struct {
	ID                int64         `xorm:"<- id AUTOINCR"`
	Owner             string        `xorm:"owner"`
	EbicsSubscriberID int64         `xorm:"ebics_subscriber_id"`

	Status              string        `xorm:"status"`
	CurrentStep         string        `xorm:"current_step"`
	IniOperationID      sql.NullInt64 `xorm:"ini_operation_id"`
	HiaOperationID      sql.NullInt64 `xorm:"hia_operation_id"`
	H3KOperationID      sql.NullInt64 `xorm:"h3k_operation_id"`
	LetterGeneratedAt   time.Time     `xorm:"letter_generated_at DATETIME(6) UTC"`
	LetterConfirmedAt   time.Time     `xorm:"letter_confirmed_at DATETIME(6) UTC"`
	BankActivationAt    time.Time     `xorm:"bank_activation_at DATETIME(6) UTC"`
	Operator            string        `xorm:"operator"`
	Reason              string        `xorm:"reason"`
	BankFeedback        string        `xorm:"bank_feedback"`
	Evidence            string        `xorm:"evidence"`
	CreatedAt           time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt           time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

### Methodes cibles

```go
func (*EbicsInitializationWorkflow) TableName() string
func (*EbicsInitializationWorkflow) Appellation() string
func (w *EbicsInitializationWorkflow) GetID() int64
func (w *EbicsInitializationWorkflow) BeforeWrite(db database.Access) error
func (w *EbicsInitializationWorkflow) AfterRead(database.ReadAccess) error
func (w *EbicsInitializationWorkflow) AfterInsert(db database.Access) error
func (w *EbicsInitializationWorkflow) AfterUpdate(db database.Access) error
```

### Helpers prives recommandes

```go
func validateEbicsInitializationStatus(value string) error
func validateEbicsInitializationStep(value string) error
func validateEbicsInitializationCoherence(w *EbicsInitializationWorkflow) error
```

### Regles de validation attendues

- `EbicsSubscriberID` obligatoire;
- `Status` borne;
- `CurrentStep` borne;
- coherence entre `CurrentStep` et presence eventuelle des operation IDs;
- impossibilite de marquer une activation banque sans evidence associee;
- impossibilite de sauter directement d'un etat initial a `ACTIVATED`.

## 5. Signature state

## 5.1 Positionnement

`signatureState` reste un sous-etat logique porte par `EbicsOperation`.

Pour la `Phase D`, la recommandation est:

- le garder encore dans `EbicsOperation.Metadata` ou dans une structure runtime
  dediee si l'on veut minimiser l'impact schema;
- mais le traiter comme une valeur controlee et stable, pas comme une cle libre.

## 5.2 Valeurs cibles

```go
const (
	SignatureStateNotApplicable              = "NOT_APPLICABLE"
	SignatureStateUnknown                    = "SIGNATURES_UNKNOWN"
	SignatureStateWaiting                    = "WAITING_SIGNATURES"
	SignatureStatePartiallyAvailable         = "SIGNATURE_PARTIALLY_AVAILABLE"
	SignatureStateReady                      = "SIGNATURE_READY"
	SignatureStateAdded                      = "SIGNATURE_ADDED"
	SignatureStateCancelled                  = "SIGNATURE_CANCELLED"
	SignatureStateRejected                   = "SIGNATURE_REJECTED"
	SignatureStateInvalid                    = "SIGNATURE_INVALID"
)
```

## 6. Runtime `pkg/protocols/modules/ebics/runtime`

## 6.1 `key_lifecycle_runner.go`

### Struct cible

```go
package runtime

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type KeyLifecycleAction struct {
	Action   string
	Operator string
	Reason   string
	Evidence map[string]any
}
```

### Interfaces cibles

```go
type KeyLifecycleStore interface {
	GetKeyLifecycleByID(id int64) (*model.EbicsKeyLifecycle, error)
	UpdateKeyLifecycle(l *model.EbicsKeyLifecycle) error
}

type CredentialGuard interface {
	CheckCredentialUsable(id int64) error
}
```

### Signatures cibles

```go
func ApplyKeyLifecycleAction(
	lifecycle *model.EbicsKeyLifecycle,
	action KeyLifecycleAction,
) error

func CanApplyKeyLifecycleAction(
	lifecycle *model.EbicsKeyLifecycle,
	action string,
) error
```

## 6.2 `initialization_runner.go`

### Struct cible

```go
package runtime

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type InitializationAction struct {
	Action   string
	Operator string
	Reason   string
	Evidence map[string]any
}
```

### Interfaces cibles

```go
type InitializationStore interface {
	GetInitializationByID(id int64) (*model.EbicsInitializationWorkflow, error)
	UpdateInitialization(w *model.EbicsInitializationWorkflow) error
}
```

### Signatures cibles

```go
func ApplyInitializationAction(
	workflow *model.EbicsInitializationWorkflow,
	action InitializationAction,
) error

func CanApplyInitializationAction(
	workflow *model.EbicsInitializationWorkflow,
	action string,
) error
```

## 6.3 `signature_state.go`

### Signatures cibles

```go
func DeriveSignatureState(
	orderType string,
	technicalCode string,
	businessCode string,
	metadata map[string]any,
) (string, error)

func IsSignatureOrder(orderType string) bool
```

### Intention

- aucune logique metier;
- uniquement derivation technique a partir de l'ordre, des return codes et des
  informations disponibles.

## 7. DTO API `pkg/admin/rest/api`

## 7.1 `ebics_key_lifecycles.go`

### Structs cibles

```go
type OutEbicsKeyLifecycle struct {
	ID                 int64   `json:"id" yaml:"id"`
	KeyUsage           string  `json:"keyUsage" yaml:"keyUsage"`
	RotationType       string  `json:"rotationType" yaml:"rotationType"`
	Status             string  `json:"status" yaml:"status"`
	CurrentCredentialID int64  `json:"currentCredentialId" yaml:"currentCredentialId"`
	NextCredentialID   *int64  `json:"nextCredentialId,omitempty" yaml:"nextCredentialId,omitempty"`
	TriggerOperationID *int64  `json:"triggerOperationId,omitempty" yaml:"triggerOperationId,omitempty"`
	LastOperationID    *int64  `json:"lastOperationId,omitempty" yaml:"lastOperationId,omitempty"`
	RequestedAt        *time.Time `json:"requestedAt,omitempty" yaml:"requestedAt,omitempty"`
	SentAt             *time.Time `json:"sentAt,omitempty" yaml:"sentAt,omitempty"`
	ActivatedAt        *time.Time `json:"activatedAt,omitempty" yaml:"activatedAt,omitempty"`
	RetiredAt          *time.Time `json:"retiredAt,omitempty" yaml:"retiredAt,omitempty"`
	Operator           string  `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason             string  `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type InEbicsKeyLifecycleAction struct {
	Action   string         `json:"action" yaml:"action"`
	Operator string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}
```

## 7.2 `ebics_initializations.go`

### Structs cibles

```go
type OutEbicsInitializationWorkflow struct {
	ID               int64      `json:"id" yaml:"id"`
	Status           string     `json:"status" yaml:"status"`
	CurrentStep      string     `json:"currentStep" yaml:"currentStep"`
	IniOperationID   *int64     `json:"iniOperationId,omitempty" yaml:"iniOperationId,omitempty"`
	HiaOperationID   *int64     `json:"hiaOperationId,omitempty" yaml:"hiaOperationId,omitempty"`
	H3KOperationID   *int64     `json:"h3kOperationId,omitempty" yaml:"h3kOperationId,omitempty"`
	LetterGeneratedAt *time.Time `json:"letterGeneratedAt,omitempty" yaml:"letterGeneratedAt,omitempty"`
	LetterConfirmedAt *time.Time `json:"letterConfirmedAt,omitempty" yaml:"letterConfirmedAt,omitempty"`
	BankActivationAt *time.Time `json:"bankActivationAt,omitempty" yaml:"bankActivationAt,omitempty"`
	Operator         string     `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason           string     `json:"reason,omitempty" yaml:"reason,omitempty"`
	BankFeedback     string     `json:"bankFeedback,omitempty" yaml:"bankFeedback,omitempty"`
}

type InEbicsInitializationAction struct {
	Action   string         `json:"action" yaml:"action"`
	Operator string         `json:"operator,omitempty" yaml:"operator,omitempty"`
	Reason   string         `json:"reason,omitempty" yaml:"reason,omitempty"`
	Evidence map[string]any `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}
```

## 7.3 `ebics_operations.go`

### Extension cible

`OutEbicsOperation` doit pouvoir exposer, quand applicable:

```go
SignatureState string `json:"signatureState,omitempty" yaml:"signatureState,omitempty"`
```

## 8. Transitions minimales a figer

## 8.1 `EbicsKeyLifecycle`

Transitions recommandees:

- `DRAFT -> MATERIAL_PREPARED`
- `MATERIAL_PREPARED -> ORDER_PLANNED`
- `ORDER_PLANNED -> ORDER_SENT`
- `ORDER_SENT -> WAITING_BANK_CONFIRMATION`
- `WAITING_BANK_CONFIRMATION -> ACTIVATED`
- `ACTIVATED -> RETIRED`
- `DRAFT -> CANCELLED`
- `MATERIAL_PREPARED -> CANCELLED`
- `ORDER_SENT -> REJECTED`

Transitions a interdire par defaut:

- `DRAFT -> ACTIVATED`
- `ORDER_PLANNED -> RETIRED`
- `REJECTED -> ACTIVATED`

## 8.2 `EbicsInitializationWorkflow`

Transitions recommandees:

- `DRAFT -> INI_PLANNED`
- `INI_PLANNED -> INI_SENT`
- `INI_SENT -> HIA_PLANNED`
- `HIA_PLANNED -> HIA_SENT`
- `HIA_SENT -> WAITING_LETTER_CONFIRMATION`
- `WAITING_LETTER_CONFIRMATION -> WAITING_BANK_ACTIVATION`
- `WAITING_BANK_ACTIVATION -> ACTIVATED`
- `ANY_ACTIVE_STATE -> CANCELLED`

Transitions a interdire par defaut:

- `DRAFT -> ACTIVATED`
- `INI_SENT -> ACTIVATED`
- `WAITING_LETTER_CONFIRMATION -> ACTIVATED` sans confirmation banque

## 9. Point de vigilance principal

La `Phase D` ne doit pas laisser subsister des transitions implicites.

Donc:

- etats nommes;
- actions nommees;
- evidences nommees;
- refus explicites;
- et exposition lisible pour l'exploitation.
