# Design detaille - `EbicsPayloadProfile`

## 1. Objet

Ce document detaille le modele exact de `EbicsPayloadProfile`:

- SQL;
- struct `model`;
- DTO REST;
- commandes d'administration CLI.

Il sert de reference pour rendre les profils payload EBICS administrables dans
Gateway.

## 2. Positionnement

`EbicsPayloadProfile` est un objet de configuration fonctionnelle
protocolaire.

Il ne remplace pas:

- `Rule`, qui reste une politique technique Gateway;
- `ContractView`, qui reste la vue constatee du contrat publie par la banque;
- `Transfer`, qui reste l'objet d'execution fichier.

Il porte une configuration reusable de soumission/collecte payload EBICS.

La vue contractuelle observee reste separee:

- `ebics_contract_views` pour l'entete de snapshot;
- `ebics_contract_view_items` pour les lignes exploitables.

Consequence importante:

- un profil peut etre concu pour ne contenir que des combinaisons deja
  contractuellement valides;
- son usage rend la commande plus sure que la saisie libre champ par champ.

## 3. Modele SQL cible

### 3.1 Table `ebics_payload_profiles`

Le bloc ci-dessous est un DDL logique de reference.

Il devra etre adapte proprement via l'ORM et les migrations Gateway pour
rester compatible multi-SGBD.

```sql
CREATE TABLE ebics_payload_profiles (
    id                          BIGINT       NOT NULL AUTO_INCREMENT,
    owner                       VARCHAR(100) NOT NULL,
    name                        VARCHAR(100) NOT NULL,
    label                       VARCHAR(150) NOT NULL DEFAULT '',
    description                 TEXT         NOT NULL DEFAULT '',
    order_type                  VARCHAR(10)  NOT NULL,
    direction                   VARCHAR(20)  NOT NULL,
    service_name                VARCHAR(40)  NOT NULL DEFAULT '',
    service_option              VARCHAR(40)  NOT NULL DEFAULT '',
    scope                       VARCHAR(40)  NOT NULL DEFAULT '',
    msg_name                    VARCHAR(100) NOT NULL DEFAULT '',
    container_type              VARCHAR(40)  NOT NULL DEFAULT '',
    default_rule_id             BIGINT,
    default_target_directory    VARCHAR(255) NOT NULL DEFAULT '',
    requires_declared_amount    BOOLEAN      NOT NULL DEFAULT FALSE,
    default_currency            VARCHAR(3)   NOT NULL DEFAULT '',
    allowed_extensions          TEXT         NOT NULL DEFAULT '[]',
    filename_pattern            VARCHAR(255) NOT NULL DEFAULT '',
    strict_contract_check       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_enabled                  BOOLEAN      NOT NULL DEFAULT TRUE,
    metadata                    TEXT         NOT NULL DEFAULT '{}',
    created_at                  DATETIME     NOT NULL,
    updated_at                  DATETIME     NOT NULL,

    CONSTRAINT ebics_payload_profiles_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_payload_profile UNIQUE (owner, name),
    CONSTRAINT ebics_payload_profile_rule_fkey FOREIGN KEY (default_rule_id)
        REFERENCES rules (id) ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT ebics_payload_profile_order_check CHECK (
        order_type IN ('BTU', 'BTD', 'FUL', 'FDL')
    ),
    CONSTRAINT ebics_payload_profile_direction_check CHECK (
        direction IN ('UPLOAD', 'DOWNLOAD', 'BIDIRECTIONAL')
    )
);
```

Indexes recommandes:

- `(owner, order_type, is_enabled)`
- `(owner, direction, is_enabled)`
- `(owner, service_name, service_option, scope, msg_name)`

## 4. Struct `model`

Struct cible recommandee:

```go
type EbicsPayloadProfile struct {
    ID                     int64          `xorm:"<- id AUTOINCR"`
    Owner                  string         `xorm:"owner"`
    Name                   string         `xorm:"name"`
    Label                  string         `xorm:"label"`
    Description            string         `xorm:"description"`
    OrderType              string         `xorm:"order_type"`
    Direction              string         `xorm:"direction"`
    ServiceName            string         `xorm:"service_name"`
    ServiceOption          string         `xorm:"service_option"`
    Scope                  string         `xorm:"scope"`
    MsgName                string         `xorm:"msg_name"`
    ContainerType          string         `xorm:"container_type"`
    DefaultRuleID          sql.NullInt64  `xorm:"default_rule_id"`
    DefaultTargetDirectory string         `xorm:"default_target_directory"`
    RequiresDeclaredAmount bool           `xorm:"requires_declared_amount"`
    DefaultCurrency        string         `xorm:"default_currency"`
    AllowedExtensions      string         `xorm:"allowed_extensions"`
    FilenamePattern        string         `xorm:"filename_pattern"`
    StrictContractCheck    bool           `xorm:"strict_contract_check"`
    IsEnabled              bool           `xorm:"is_enabled"`
    Metadata               string         `xorm:"metadata"`
    CreatedAt              time.Time      `xorm:"created_at DATETIME(6) UTC"`
    UpdatedAt              time.Time      `xorm:"updated_at DATETIME(6) UTC"`
}
```

Note importante:

- pour rester cohérent avec la contrainte multi-SGBD, `AllowedExtensions` et
  `Metadata` sont de preference stockes en `model` sous forme serialisee;
- la conversion vers `[]string` et `map[string]any` reste pertinente au niveau
  DTO REST et logique applicative.

## 5. DTO REST cibles

### 5.1 `api.OutEbicsPayloadProfile`

```go
type OutEbicsPayloadProfile struct {
    ID                     int64            `json:"id" yaml:"id"`
    Name                   string           `json:"name" yaml:"name"`
    Label                  string           `json:"label,omitempty" yaml:"label,omitempty"`
    Description            string           `json:"description,omitempty" yaml:"description,omitempty"`
    OrderType              string           `json:"orderType" yaml:"orderType"`
    Direction              string           `json:"direction" yaml:"direction"`
    ServiceName            string           `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
    ServiceOption          string           `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
    Scope                  string           `json:"scope,omitempty" yaml:"scope,omitempty"`
    MsgName                string           `json:"msgName,omitempty" yaml:"msgName,omitempty"`
    ContainerType          string           `json:"containerType,omitempty" yaml:"containerType,omitempty"`
    DefaultRule            Nullable[string] `json:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
    DefaultTargetDirectory string           `json:"defaultTargetDirectory,omitempty" yaml:"defaultTargetDirectory,omitempty"`
    RequiresDeclaredAmount bool             `json:"requiresDeclaredAmount" yaml:"requiresDeclaredAmount"`
    DefaultCurrency        string           `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
    AllowedExtensions      []string         `json:"allowedExtensions,omitempty" yaml:"allowedExtensions,omitempty"`
    FilenamePattern        string           `json:"filenamePattern,omitempty" yaml:"filenamePattern,omitempty"`
    StrictContractCheck    bool             `json:"strictContractCheck" yaml:"strictContractCheck"`
    IsEnabled              bool             `json:"isEnabled" yaml:"isEnabled"`
    Metadata               map[string]any   `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
```

### 5.2 `api.InEbicsPayloadProfile`

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
```

## 6. Routes REST cibles

Routes recommandees:

- `GET /api/ebics/payload-profiles`
- `POST /api/ebics/payload-profiles`
- `GET /api/ebics/payload-profiles/{profile}`
- `PATCH /api/ebics/payload-profiles/{profile}`

Validation a l'execution:

- le tuple `orderType/serviceName/serviceOption/scope/msgName` du profil est
  verifie contre la vue contractuelle active;
- `strictContractCheck=true` impose un rejet si aucune ligne compatible n'est
  trouvee dans `ebics_contract_view_items`.

Recommendation de gouvernance:

- en exploitation courante, n'autoriser que des profils valides
  contractuellement;
- reserver la saisie libre aux administrateurs, au diagnostic et au bootstrap;
- si un profil devient invalide contractuellement apres evolution du contrat
  banque, le desactiver ou le rejeter a l'execution.

## 7. Commandes CLI d'administration

Commandes cibles:

```text
waarp-gateway ebics payload profile add
waarp-gateway ebics payload profile get <name>
waarp-gateway ebics payload profile list
waarp-gateway ebics payload profile update <name>
```

Options de `add` minimales:

- `--name`
- `--label`
- `--description`
- `--order-type`
- `--direction`
- `--service-name`
- `--service-option`
- `--scope`
- `--msg-name`
- `--container`
- `--default-rule`
- `--default-target-dir`
- `--requires-declared-amount`
- `--default-currency`
- `--allowed-extension`
- `--filename-pattern`
- `--strict-contract-check`
- `--disable`

## 8. Articulation avec `updateconf`

`EbicsPayloadProfile` doit etre:

- exportable;
- importable;
- serialisable en JSON/YAML;
- pris en charge par `updateconf`.

## 9. Securisation de la commande

L'usage du profil doit etre considere comme plus sur que la commande libre,
parce que:

- le profil est relu, revu et gouverne;
- il peut etre verifie a froid contre le contrat connu;
- il reduit fortement les erreurs de saisie;
- il limite la creation d'ordres protocolaires incoherents.

L'API et la CLI devraient donc a terme supporter une politique telle que:

- `profile-required` pour les environnements de production tres gouvernes;
- `profile-preferred` comme mode par defaut;
- `free-input-allowed` seulement pour les usages d'administration avances.
