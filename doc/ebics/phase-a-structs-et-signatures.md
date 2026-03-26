# Phase A - Structs Go et signatures de methodes

## 1. Objet

Ce document pousse la `Phase A` jusqu'au niveau des declarations Go cibles.

L'objectif est de figer:

- les structs;
- les constantes;
- les signatures de methodes;
- les responsabilites de chaque fichier;
- les points d'extension autorises.

Il ne s'agit pas encore de fournir le code complet, mais de rendre la phase
directement implementable sans redecision d'architecture.

## 2. Rappels de conventions Gateway

Les propositions ci-dessous s'alignent sur les patterns observes dans:

- [client.go](c:\MonProjet\Waarp-Gateway\pkg\model\client.go)
- [local_agent.go](c:\MonProjet\Waarp-Gateway\pkg\model\local_agent.go)
- [credentials.go](c:\MonProjet\Waarp-Gateway\pkg\model\credentials.go)
- [protocol.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\protocol.go)
- [services.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\protocol\services.go)
- [config.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\protocol\config.go)

Regles a conserver:

- `BeforeWrite(db database.Access) error` pour les validations model;
- `TableName() string`, `Appellation() string`, `GetID() int64`;
- `ProtoConfig` sous forme `map[string]any` cote `Client`, `LocalAgent`,
  `RemoteAgent`;
- structs `serverConfig`, `clientConfig`, `partnerConfig` cote protocole;
- interfaces `protocol.Server` et `protocol.Client` respectees strictement.

Regle supplementaire importante:

- les structs persistentes doivent rester compatibles avec le multi-SGBD
  Gateway via XORM, donc sans type vendor-specifique ni dependance a des
  comportements propres a un moteur SQL particulier.

## 3. Constantes model

## 3.1 `pkg/model/table_names.go`

Ajouts cibles:

```go
const (
	TableEbicsHosts       = "ebics_hosts"
	TableEbicsSubscribers = "ebics_subscribers"
	TableEbicsBankKeys    = "ebics_bank_keys"
)
```

## 3.2 `pkg/model/display_names.go`

Ajouts cibles:

```go
const (
	NameEbicsHost       = "ebics host"
	NameEbicsSubscriber = "ebics subscriber"
	NameEbicsBankKey    = "ebics bank key"
)
```

## 4. Models `pkg/model`

## 4.1 `pkg/model/ebics_host.go`

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

type EbicsHost struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	Name        string `xorm:"name"`
	HostID      string `xorm:"host_id"`
	Description string `xorm:"description"`

	Enabled         bool   `xorm:"enabled"`
	IsServer        bool   `xorm:"is_server"`
	ProtocolVersion string `xorm:"protocol_version"`
	Transport       string `xorm:"transport"`
	DefaultBankURL  string `xorm:"default_bank_url"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

Note de conception:

- rester sur des types simples (`string`, `bool`, `time.Time`, `int64`);
- pas de champ `map[string]any` ou structure libre dans les objets `Phase A`
  tant qu'il n'y a pas de besoin strict;
- pas de dependance a un type JSON natif SGBD.

### Methodes cibles

```go
func (*EbicsHost) TableName() string
func (*EbicsHost) Appellation() string
func (h *EbicsHost) GetID() int64
func (h *EbicsHost) BeforeWrite(db database.Access) error
```

### Squelette de validation recommande

```go
func (h *EbicsHost) BeforeWrite(db database.Access) error {
	h.Owner = conf.GlobalConfig.GatewayName

	h.Name = strings.TrimSpace(h.Name)
	h.HostID = strings.TrimSpace(h.HostID)
	h.ProtocolVersion = strings.TrimSpace(h.ProtocolVersion)
	h.Transport = strings.TrimSpace(h.Transport)
	h.DefaultBankURL = strings.TrimSpace(h.DefaultBankURL)

	if h.Name == "" {
		h.Name = h.HostID
	}

	if h.Name == "" {
		return database.NewValidationError("the EBICS host name cannot be empty")
	}

	if h.HostID == "" {
		return database.NewValidationError("the EBICS host ID is missing")
	}

	if err := validateEbicsProtocolVersion(h.ProtocolVersion); err != nil {
		return err
	}

	if err := validateEbicsTransport(h.Transport); err != nil {
		return err
	}

	if n, err := db.Count(h).Where("id<>? AND owner=? AND name=?", h.ID, h.Owner, h.Name).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS hosts by name: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS host named %q already exists", h.Name)
	}

	if n, err := db.Count(h).Where("id<>? AND owner=? AND host_id=?", h.ID, h.Owner, h.HostID).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS hosts by host ID: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS host with host ID %q already exists", h.HostID)
	}

	return nil
}
```

### Helpers prives autorises

```go
func validateEbicsProtocolVersion(version string) error
func validateEbicsTransport(transport string) error
```

## 4.2 `pkg/model/ebics_subscriber.go`

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

type EbicsSubscriber struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsHostID int64 `xorm:"ebics_host_id"`

	Name      string `xorm:"name"`
	PartnerID string `xorm:"partner_id"`
	UserID    string `xorm:"user_id"`
	SystemID  string `xorm:"system_id"`

	TransportURL             string `xorm:"transport_url"`
	Enabled                  bool   `xorm:"enabled"`
	DefaultOrderDataEncoding string `xorm:"default_order_data_encoding"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

Note de conception:

- meme logique de simplicite et de portabilite ORM;
- pas de colonnes semi-structurees dans `Phase A`.

### Methodes cibles

```go
func (*EbicsSubscriber) TableName() string
func (*EbicsSubscriber) Appellation() string
func (s *EbicsSubscriber) GetID() int64
func (s *EbicsSubscriber) BeforeWrite(db database.Access) error
```

### Squelette de validation recommande

```go
func (s *EbicsSubscriber) BeforeWrite(db database.Access) error {
	s.Owner = conf.GlobalConfig.GatewayName

	s.Name = strings.TrimSpace(s.Name)
	s.PartnerID = strings.TrimSpace(s.PartnerID)
	s.UserID = strings.TrimSpace(s.UserID)
	s.SystemID = strings.TrimSpace(s.SystemID)
	s.TransportURL = strings.TrimSpace(s.TransportURL)
	s.DefaultOrderDataEncoding = strings.TrimSpace(s.DefaultOrderDataEncoding)

	if s.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if s.PartnerID == "" {
		return database.NewValidationError("the EBICS partner ID is missing")
	}

	if s.UserID == "" {
		return database.NewValidationError("the EBICS user ID is missing")
	}

	if s.Name == "" {
		s.Name = s.PartnerID + ":" + s.UserID
	}

	var host EbicsHost
	if err := db.Get(&host, "id=?", s.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", s.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	if n, err := db.Count(s).Where(
		"id<>? AND owner=? AND ebics_host_id=? AND partner_id=? AND user_id=?",
		s.ID, s.Owner, s.EbicsHostID, s.PartnerID, s.UserID,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS subscribers: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS subscriber already exists for partner %q and user %q", s.PartnerID, s.UserID)
	}

	return nil
}
```

## 4.3 `pkg/model/ebics_bank_key.go`

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

type EbicsBankKey struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsHostID int64 `xorm:"ebics_host_id"`

	KeyType       string `xorm:"key_type"`
	Version       string `xorm:"version"`
	PublicKey     string `xorm:"public_key"`
	PublicKeyHash string `xorm:"public_key_hash"`
	State         string `xorm:"state"`

	ValidFrom time.Time `xorm:"valid_from DATETIME(6) UTC"`
	ValidTo   time.Time `xorm:"valid_to DATETIME(6) UTC"`
	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}
```

Note de conception:

- la cle publique reste ici sous forme `string` portable;
- toute optimisation de stockage plus specialisee est explicitement hors
  `Phase A`.

### Methodes cibles

```go
func (*EbicsBankKey) TableName() string
func (*EbicsBankKey) Appellation() string
func (k *EbicsBankKey) GetID() int64
func (k *EbicsBankKey) BeforeWrite(db database.Access) error
```

### Helpers prives autorises

```go
func validateEbicsBankKeyType(keyType string) error
func validateEbicsBankKeyState(state string) error
```

### Squelette de validation recommande

```go
func (k *EbicsBankKey) BeforeWrite(db database.Access) error {
	k.Owner = conf.GlobalConfig.GatewayName

	k.KeyType = strings.TrimSpace(k.KeyType)
	k.Version = strings.TrimSpace(k.Version)
	k.PublicKey = strings.TrimSpace(k.PublicKey)
	k.PublicKeyHash = strings.TrimSpace(k.PublicKeyHash)
	k.State = strings.TrimSpace(k.State)

	if k.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if err := validateEbicsBankKeyType(k.KeyType); err != nil {
		return err
	}

	if err := validateEbicsBankKeyState(k.State); err != nil {
		return err
	}

	if k.PublicKey == "" {
		return database.NewValidationError("the EBICS bank public key is missing")
	}

	var host EbicsHost
	if err := db.Get(&host, "id=?", k.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", k.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	if n, err := db.Count(k).Where(
		"id<>? AND owner=? AND ebics_host_id=? AND key_type=? AND version=?",
		k.ID, k.Owner, k.EbicsHostID, k.KeyType, k.Version,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS bank keys: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS bank key already exists for type %q and version %q", k.KeyType, k.Version)
	}

	return nil
}
```

## 5. Module protocolaire `pkg/protocols/modules/ebics`

## 5.1 `constants.go`

### Contenu cible

```go
package ebics

const (
	EBICS = "ebics"
)

const (
	transportHTTPS = "https"
)

const (
	profileRequired   = "profile-required"
	profilePreferred  = "profile-preferred"
	freeInputAllowed  = "free-input-allowed"
)

const (
	protocolVersionH004 = "H004"
	protocolVersionH005 = "H005"
)
```

### Eventuels types aliases acceptables

```go
type profilePolicy string
type protocolVersion string
```

## 5.2 `errors.go`

### Variables d'erreur cibles

```go
package ebics

import "errors"

var (
	ErrInvalidConfig          = errors.New("invalid EBICS configuration")
	ErrUnsupportedTransport   = errors.New("unsupported EBICS transport")
	ErrUnsupportedVersion     = errors.New("unsupported EBICS protocol version")
	ErrNotImplemented         = errors.New("EBICS feature not implemented yet")
	ErrSubscriberNotFound     = errors.New("EBICS subscriber not found")
	ErrHostNotFound           = errors.New("EBICS host not found")
	ErrBankKeyVerificationOff = errors.New("EBICS bank key verification is disabled")
)
```

### Helpers recommandes

```go
func wrapConfigError(err error) error
```

## 5.3 `config.go`

### Structs cibles

```go
package ebics

import (
	"fmt"
	"strings"
	"time"
)

type serverConfig struct {
	ProtocolVersion string        `json:"protocolVersion,omitempty"`
	RequestTimeout  time.Duration `json:"requestTimeout,omitempty"`
	MaxSegmentSize  int64         `json:"maxSegmentSize,omitempty"`
	AllowRecovery   bool          `json:"allowRecovery,omitempty"`
	TLSMinVersion   string        `json:"tlsMinVersion,omitempty"`
	VerifyBankKeys  bool          `json:"verifyBankKeys,omitempty"`
}

type clientConfig struct {
	ProtocolVersion          string        `json:"protocolVersion,omitempty"`
	EndpointURL              string        `json:"endpointURL,omitempty"`
	RequestTimeout           time.Duration `json:"requestTimeout,omitempty"`
	MaxSegmentSize           int64         `json:"maxSegmentSize,omitempty"`
	AllowRecovery            bool          `json:"allowRecovery,omitempty"`
	TLSMinVersion            string        `json:"tlsMinVersion,omitempty"`
	VerifyBankKeys           bool          `json:"verifyBankKeys,omitempty"`
	DefaultOrderDataEncoding string        `json:"defaultOrderDataEncoding,omitempty"`
	ProfilePolicy            string        `json:"profilePolicy,omitempty"`
}

type partnerConfig struct {
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	EndpointURL     string `json:"endpointURL,omitempty"`
	HostID          string `json:"hostID,omitempty"`
	UseWSSRTN       bool   `json:"useWSSRTN,omitempty"`
}
```

### Signatures cibles

```go
func (c *serverConfig) ValidServer() error
func (c *clientConfig) ValidClient() error
func (c *partnerConfig) ValidPartner() error
```

### Helpers prives recommandes

```go
func normalizeProtocolVersion(version string) string
func validateProtocolVersion(version string) error
func validateTLSMinVersion(version string) error
func validateProfilePolicy(policy string) error
func validateEndpointURL(rawURL string) error
func defaultServerConfig() *serverConfig
func defaultClientConfig() *clientConfig
func defaultPartnerConfig() *partnerConfig
```

### Ligne d'implementation

- configuration vide valide par defaut;
- normalisation dans `Valid*`;
- aucune resolution depuis la base a ce niveau;
- aucune semantique `contract view` a ce niveau.
- rester sur des types JSON/YAML naturellement serialisables et deja bien
  supportes par l'existant.

## 5.4 `module.go`

### Struct et signatures cibles

```go
package ebics

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type Module struct{}

func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server
func (Module) NewClient(db *database.DB, client *model.Client) protocol.Client
func (Module) MakeServerConfig() protocol.ServerConfig
func (Module) MakeClientConfig() protocol.ClientConfig
func (Module) MakePartnerConfig() protocol.PartnerConfig
```

### Corps attendus

```go
func (Module) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return NewServer(db, server)
}

func (Module) NewClient(db *database.DB, client *model.Client) protocol.Client {
	return NewClient(db, client)
}

func (Module) MakeServerConfig() protocol.ServerConfig {
	return defaultServerConfig()
}

func (Module) MakeClientConfig() protocol.ClientConfig {
	return defaultClientConfig()
}

func (Module) MakePartnerConfig() protocol.PartnerConfig {
	return defaultPartnerConfig()
}
```

## 5.5 `server.go`

### Struct cible

```go
package ebics

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Server struct {
	db     *database.DB
	logger *log.Logger
	server *model.LocalAgent
	config *serverConfig
}
```

### Signatures cibles

```go
func NewServer(db *database.DB, dbServer *model.LocalAgent) *Server
func (s *Server) Start() error
func (s *Server) Stop(ctx context.Context) error
func (s *Server) State() (utils.StateCode, string)
```

### Helpers prives recommandes

```go
func loadServerConfig(server *model.LocalAgent) (*serverConfig, error)
```

### Ligne d'implementation attendue

- `NewServer` ne doit pas panic;
- `Start` valide la configuration et renvoie une erreur explicite tant que le
  serveur EBICS complet n'est pas livre;
- `Stop` doit etre idempotent;
- `State` doit renvoyer un etat stable et un message lisible.

## 5.6 `client.go`

### Structs cibles

```go
package ebics

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Client struct {
	db     *database.DB
	logger *log.Logger
	client *model.Client
	config *clientConfig
}

type transferClient struct{}
```

### Signatures cibles

```go
func NewClient(db *database.DB, dbClient *model.Client) *Client
func (c *Client) Start() error
func (c *Client) Stop(ctx context.Context) error
func (c *Client) State() (utils.StateCode, string)
func (c *Client) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error)
```

### Helpers prives recommandes

```go
func loadClientConfig(client *model.Client) (*clientConfig, error)
```

### Ligne d'implementation attendue

- `Start` et `Stop` doivent etre coherents meme si le client est surtout
  “on demand”;
- `InitTransfer` peut retourner une erreur `not implemented yet`, mais pas un
  objet partiel trompeur;
- les logs doivent porter le nom du client et le protocole.

## 6. Enregistrement du protocole

## 6.1 `pkg/protocols/modules.go`

Ajout cible:

```go
import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics"
)
```

et dans `List`:

```go
	ebics.EBICS: &ebics.Module{},
```

## 7. Points a laisser explicitement hors Phase A

Les signatures suivantes ne doivent pas encore etre posees en code si leur
perimetre n'est pas stabilise:

- stores EBICS;
- `EbicsOperation`;
- `EbicsContractView`;
- `EbicsPayloadProfile`;
- gestion RTN;
- workflow de rotation des cles;
- workflow d'initialisation;
- gestion detaillee des signatures protocolaires.

La raison est simple:

- les creer trop tot augmenterait le bruit et les faux points fixes;
- alors que `Phase A` doit rester courte, propre et durable.

## 8. Definition de done de ce niveau de detail

Le niveau de specification est suffisant si un developpeur peut:

- creer les fichiers sans reinventer les structs;
- implementer les signatures sans redecider les interfaces;
- savoir ou placer les validations;
- savoir ce qui est volontairement hors scope;
- et coder la `Phase A` sans rouvrir les choix structurants.
