# Backlog d'implementation EBICS

## 1. Objet

Ce document transforme les dossiers de cadrage en plan d'implementation
operationnel.

Il couvre:

- un backlog par lots;
- un modele SQL cible de reference;
- les surfaces REST et CLI a ajouter;
- une matrice de mapping entre objets Gateway et objets EBICS.

Il introduit aussi explicitement la notion `EbicsOperation` pour tous les
ordres EBICS non payload.

La matrice de decision par ordre EBICS fait foi pour les politiques
`Transfer/non-Transfer`, identifiants, replay et reprise.

Les routes REST detaillees EBICS font foi pour la derivation vers
`pkg/admin/rest`.

Le backlog concret de developpement fichier par fichier est detaille dans:

- `backlog-implementation-fichier-par-fichier.md`

Les checklists de suivi a cocher pendant l'implementation sont:

- `suivi-implementation-ebics.md`
- `suivi-implementation-phases.md`

## 2. Principes de mise en oeuvre

- reutiliser les conventions de Gateway avant d'ajouter de nouveaux patterns;
- isoler les invariants EBICS dans des tables et composants dedies;
- traiter la rotation des cles et RTN comme des automatismes protocolaires;
- limiter l'initialisation EBICS a un support technique avec point d'arret
  explicite, sans reconstruire un workflow bancaire complet;
- gerer les signatures au sens protocolaire, sans porter le workflow metier de
  signature;
- exposer un passe-plat vers l'application metier pour les decisions non
  protocolaires;
- reserver `Transfer` aux flux reellement orientes fichier;
- construire d'abord le socle durable avant d'ajouter les parcours UI.

## 2.1 Perimetre cible de decision

Le backlog ci-dessous doit etre lu avec un perimetre volontairement recentre.

Sont dans le perimetre cible Gateway:

- socle protocolaire EBICS serveur et client;
- rotation automatisee des cles quand elle releve du protocole;
- recuperation automatisee des reports;
- notifications RTN et auto-pull associe;
- verification et collecte des signatures au sens protocolaire;
- exposition d'evenements techniques vers l'application metier.
- connecteurs standards de passe-plat vers le SI metier, y compris messaging.

Ne sont pas retenus en phase 1 comme prerequis du sujet EBICS:

- filewatcher integre au coeur Gateway;
- client lourd de poste utilisateur.

Restent hors perimetre cible Gateway:

- workflow metier de signature;
- decisions de validation ou de liberation metier;
- pilotage humain bancaire complet de l'initialisation;
- arbitrages fonctionnels non protocolaires.

## 3. Backlog par lots

## 3.1 Lot 0 - Cadrage et preparation

Objectif:

- verrouiller les choix de modele et d'API avant implementation.

Taches:

- valider le mapping `LocalAgent` / `Client` / `RemoteAgent` / `Account`;
- valider la liste initiale des ordres a supporter en P1;
- valider le modele SQL de reference;
- fixer les statuts cibles pour initialisation, rotation et RTN;
- fixer les conventions de nommage `proto_config` EBICS.

Livrables:

- ADR ou note de decision interne;
- backlog valide;
- schema cible valide.

Critere de sortie:

- plus d'ambiguite sur ce qui va dans `Transfer`, `TransferInfo` et les tables
  EBICS dediees.
- plus d'ambiguite sur les surfaces d'administration `EbicsOperation`.

## 3.1 bis Lot 0A - Protocoles AMQP Gateway

Objectif:

- introduire `AMQP 0.9.1` et `AMQP 1.0` comme protocoles Gateway natifs,
  independants d'EBICS.

Taches:

- definir les modules `pkg/protocols/modules/amqp091/` et
  `pkg/protocols/modules/amqp10/`;
- definir leurs `ProtoConfig`;
- raccorder leur cycle de vie au registre protocolaire standard;
- definir une outbox / inbox mutualisee pour les integrations metier;
- definir les contrats de message techniques et la correlation;
- ajouter une administration REST/CLI minimale.

Livrables:

- deux protocoles Gateway supplementaires exploitables;
- premiers tests de publication / consommation;
- socle d'integration asynchrone reutilisable par EBICS.

Critere de sortie:

- la Gateway peut publier et/ou consommer des messages AMQP sans dependre
  d'EBICS;
- la couche de passe-plat metier ne depend plus seulement de FS / REST / CLI.

## 3.2 Lot 1 - Socle protocolaire EBICS

Objectif:

- rendre Gateway capable de reconnaitre et demarrer le protocole `ebics`.

Taches:

- ajouter `pkg/protocols/modules/ebics/module.go`;
- introduire une notion durable `EbicsOperation` pour les ordres non payload;
- fixer une politique d'identifiants `Gateway <-> EBICS`;
- fixer une politique `retry / replay / non-replay` par type d'ordre;
- fixer la politique `resume Gateway <-> recovery / segmentation EBICS`;
- ajouter les structures `ServerConfig`, `ClientConfig`, `PartnerConfig`;
- enregistrer `ebics` dans [pkg/protocols/modules.go](c:\MonProjet\Waarp-Gateway\pkg\protocols\modules.go);
- ajouter les validateurs de `ProtoConfig`;
- ajouter le squelette `server.go` et `client.go`;
- raccorder le demarrage/arret au cycle existant de [server.go](c:\MonProjet\Waarp-Gateway\pkg\gatewayd\server.go).

Livrables:

- protocole `ebics` visible depuis les modeles et l'admin;
- distinction explicite `EbicsOperation` / `Transfer`;
- demarrage de service sans logique metier complete;
- tests de registration et de cycle de vie.

Surfaces d'administration minimales:

- ressource REST `/api/ebics/operations`;
- commandes CLI `ebics operation`;
- permissions alignees initialement sur `transfers`.

Dependances:

- lot 0 valide.

## 3.3 Lot 2 - Stores durables EBICS

Objectif:

- fournir la couche de persistance imposee par la librairie EBICS.

Taches:

- ajouter les modeles SQL pour:
  - `ebics_hosts`
  - `ebics_subscribers`
  - `ebics_bank_keys`
  - `ebics_operations`
  - `ebics_transactions`
  - `ebics_transaction_segments`
  - `ebics_nonce_window`
  - `ebics_contract_views`
- implementer les adapters Gateway vers:
  - `store.KeyStore`
  - `store.SubscriberStore`
  - `store.TxStore`
  - `store.NonceStore`
- ajouter les migrations;
- ajouter index, purge et contraintes d'unicite.

Livrables:

- stores durables operationnels;
- jeux de tests unitaires et integration SQL.

Critere de sortie:

- le serveur EBICS peut tourner sans store memoire pour transactions et nonces.
- les ordres EBICS non payload sont persistables sans `Transfer`;
- la politique de correlation d'identifiants est fermee;
- la politique de replay et de reprise EBICS est fermee;
- la vue contractuelle technique banque est persistable et exploitable pour des
  controles preventifs.

## 3.4 Lot 3 - Initialisation technique et rupture d'automatisme

Objectif:

- fournir le support technique d'initialisation EBICS sans faire de Gateway un
  moteur de workflow bancaire.

Taches:

- ajouter la table `ebics_initialization_workflows` avec un objectif de suivi
  technique et non de pilotage metier;
- ajouter des statuts minimaux de rupture d'automatisme;
- ajouter le service d'initialisation technique;
- integrer les helpers de lettre:
  - `RenderINILetter`
  - `RenderHIALetter`
  - `RenderH3KLetter`
- produire un format de sortie telechargeable;
- bloquer l'activation production tant qu'une confirmation externe n'est pas
  enregistree;
- permettre la reprise par API/CLI apres decision externe.

Livrables:

- API et CLI de creation, suivi et confirmation technique;
- preuves d'initialisation stockees;
- journalisation operateur minimale;
- interface claire avec l'application metier ou un operateur externe.

Critere de sortie:

- impossibilite de marquer un abonne actif sans etat technique coherent;
- absence de logique de decision metier dans Gateway.

## 3.5 Lot 4 - Flux fichier EBICS

Objectif:

- supporter les echanges fichier prioritaires via le pipeline Gateway.

Taches:

- connecter `FUL`, `FDL`, `BTU`, `BTD`;
- creer/enrichir `Transfer` quand un ordre manipule un fichier;
- remplir `TransferInfo` avec les metadonnees EBICS minimales;
- connecter pre/post tasks;
- historiser la relation `TransactionID` EBICS <-> `Transfer.ID`.

Livrables:

- execution de flux fichier EBICS avec historique Gateway;
- supervision unifiee.

Critere de sortie:

- un flux fichier EBICS est visible, tracable et exploitable comme les autres
  protocoles.

## 3.6 Lot 5 - Rotation de cles

Objectif:

- rendre gerable le cycle de vie cryptographique en production.

Taches:

- ajouter `ebics_key_lifecycles`;
- introduire la notion de credential futur/actif/retire;
- brancher les ordres de rotation `PUB`, `HCA`, `HCS`, `HSA`, `SPR`, `H3K`;
- ajouter les controles d'expiration et d'obsolescence;
- tracer les preuves et validations de bascule.

Livrables:

- workflow de rotation;
- surfaces REST/CLI;
- alertes et statuts d'exploitation.

Critere de sortie:

- une rotation ne se resume plus a un overwrite de credential.

## 3.7 Lot 6 - Reporting protocolaire et signatures

Objectif:

- couvrir les ordres non fichier utiles au protocole cible, en particulier les
  reports et les signatures au sens EBICS.

Taches:

- supporter `HEV`, `HPB`, `HPD`, `HTD`, `HKD`, `HAA`;
- ajouter les objets d'historique EBICS dedies;
- collecter et rafraichir la vue contractuelle technique issue de
  `HPD` / `HTD` / `HKD` / `HAA`;
- ajouter les commandes operateur associees;
- prevoir le support progressif de `HAC`, `HVD`, `HVE`, `HVT`, `HVU`, `HVZ`,
  `HVS`;
- exposer les etats de signature et les preuves techniques;
- ne pas implementer de circuit metier de validation ou de cosignature.

Livrables:

- historique EBICS consultable;
- collecte des reports et statuts de signature;
- vue contractuelle technique consultable;
- passe-plat vers l'application metier quand une decision est requise.

## 3.8 Lot 7 - RTN et auto-pull

Objectif:

- ajouter la capacite evenementielle RTN absente de Gateway.

Taches:

- ajouter `ebics_rtn_events`;
- implementer un composant d'ingestion RTN;
- viser en phase 1 un provider de transport `WebSocket/WSS`;
- utiliser `ebics/rtnspec` ou `provider/rtn` pour parsing et mapping;
- brancher un `EventStore` durable;
- resoudre les plans de pull;
- piloter les modes `manual`, `auto`, `auto_filtered`;
- lier les pulls declenches aux historiques RTN.

Livrables:

- intake RTN;
- replay et idempotence;
- derivation detaillee des routes REST EBICS en commandes CLI, avec separation
  explicite des scopes de return codes;
- definition explicite des commandes et routes de soumission payload
  `BTU/BTD/FUL/FDL`, en projection `EbicsOperation + Transfer`;
- introduction d'une notion `EbicsPayloadProfile` pour simplifier la
  soumission des ordres payload sans surcharger `Rule`;
- modelisation complete de `EbicsPayloadProfile` en SQL, REST, CLI et
  `updateconf`;
- definir une politique d'usage `profile-required | profile-preferred |
  free-input-allowed` pour la soumission payload EBICS;
- formaliser les DTO de soumission payload et la resolution
  `explicite > profile > defaults > validation contractuelle`;
- auto-pull EBICS pilote.

Critere de sortie:

- une notification RTN peut declencher un ou plusieurs pulls standards sans
  perte de tracabilite.

## 3.9 Lot 8 - Administration avancee et UI

Objectif:

- rendre l'ensemble exploitable sans passer uniquement par REST brut.

Taches:

- ajouter les routes UI;
- ajouter la vue `Operations EBICS`;
- ajouter les ecrans de workflow d'initialisation;
- ajouter les ecrans de rotation de cles;
- ajouter la supervision RTN;
- ajouter les vues de correlation ordres/transferts.

Livrables:

- UI v1 d'exploitation EBICS.

## 3.10 Lot 9 - Integration EBICS sur socle AMQP

Objectif:

- brancher EBICS sur un socle messaging deja natif dans Gateway.

Taches:

- raccorder les evenements techniques EBICS aux protocoles `AMQP 0.9.1` et
  `AMQP 1.0`;
- definir les payloads de reference pour:
  - evenement technique sortant;
  - commande d'execution entrante;
  - accuse et echec de livraison;
- traiter correlation, retry et dead-letter dans le contexte EBICS;
- valider le modele de decouplage SI metier <-> Gateway.

Livrables:

- integration EBICS exploitable sur socle AMQP;
- supervision de livraison;
- runbooks d'exploitation.

Critere de sortie:

- un evenement technique EBICS peut etre publie de maniere fiable vers un bus
  AMQP sans couplage au code metier;
- une commande deja decidee par le metier peut revenir a Gateway via AMQP.

## 3.11 Lot 10 - Industrialisation

Objectif:

- rendre l'integration defendable en production.

Taches:

- purge et retention;
- dashboards et alertes;
- campagnes d'interop et de charge;
- runbooks d'exploitation;
- documentation admin/utilisateur.

## 4. Modele SQL cible de reference

Le schema ci-dessous est un schema logique de reference aligne avec les besoins
identifies. Il n'est pas necessairement le schema physique final.

## 4.1 Table `ebics_hosts`

```sql
CREATE TABLE ebics_hosts (
    id                  BIGINT PRIMARY KEY,
    owner               VARCHAR(100) NOT NULL,
    name                VARCHAR(100) NOT NULL,
    host_id             VARCHAR(100) NOT NULL,
    mode                VARCHAR(20)  NOT NULL,
    gateway_entity_type VARCHAR(30)  NOT NULL,
    gateway_entity_id   BIGINT       NOT NULL,
    status              VARCHAR(30)  NOT NULL,
    options             TEXT         NOT NULL DEFAULT '{}',
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL,
    CONSTRAINT unique_ebics_host UNIQUE (owner, host_id, mode)
);
```

## 4.2 Table `ebics_subscribers`

```sql
CREATE TABLE ebics_subscribers (
    id                 BIGINT PRIMARY KEY,
    owner              VARCHAR(100) NOT NULL,
    ebics_host_id      BIGINT       NOT NULL,
    partner_id         VARCHAR(100) NOT NULL,
    user_id            VARCHAR(100) NOT NULL,
    local_account_id   BIGINT,
    remote_account_id  BIGINT,
    state              VARCHAR(30)  NOT NULL,
    permissions        TEXT         NOT NULL DEFAULT '{}',
    options            TEXT         NOT NULL DEFAULT '{}',
    created_at         DATETIME     NOT NULL,
    updated_at         DATETIME     NOT NULL,
    CONSTRAINT unique_ebics_subscriber UNIQUE (owner, ebics_host_id, partner_id, user_id),
    CONSTRAINT chk_ebics_subscriber_target CHECK (
        (CASE WHEN local_account_id IS NOT NULL THEN 1 ELSE 0 END) +
        (CASE WHEN remote_account_id IS NOT NULL THEN 1 ELSE 0 END) = 1
    )
);
```

## 4.3 Table `ebics_bank_keys`

```sql
CREATE TABLE ebics_bank_keys (
    id                    BIGINT PRIMARY KEY,
    owner                 VARCHAR(100) NOT NULL,
    ebics_host_id         BIGINT       NOT NULL,
    auth_public_key       TEXT         NOT NULL DEFAULT '',
    enc_public_key        TEXT         NOT NULL DEFAULT '',
    sig_public_key        TEXT         NOT NULL DEFAULT '',
    auth_key_digest       TEXT         NOT NULL DEFAULT '',
    enc_key_digest        TEXT         NOT NULL DEFAULT '',
    sig_key_digest        TEXT         NOT NULL DEFAULT '',
    status                VARCHAR(30)  NOT NULL,
    valid_from            DATETIME,
    valid_to              DATETIME,
    created_at            DATETIME     NOT NULL,
    updated_at            DATETIME     NOT NULL
);
```

## 4.4 Table `ebics_transactions`

```sql
CREATE TABLE ebics_transactions (
    id                 BIGINT PRIMARY KEY,
    owner              VARCHAR(100) NOT NULL,
    tx_id              VARCHAR(100) NOT NULL,
    ebics_host_id      BIGINT       NOT NULL,
    ebics_subscriber_id BIGINT      NOT NULL,
    order_type         VARCHAR(20)  NOT NULL,
    transfer_id        BIGINT,
    status             VARCHAR(30)  NOT NULL,
    segment_count      INTEGER      NOT NULL DEFAULT 0,
    recovery_counter   INTEGER      NOT NULL DEFAULT 0,
    recovery_point     VARCHAR(255) NOT NULL DEFAULT '',
    created_at         DATETIME     NOT NULL,
    updated_at         DATETIME     NOT NULL,
    CONSTRAINT unique_ebics_tx UNIQUE (owner, tx_id)
);
```

## 4.5 Table `ebics_transaction_segments`

```sql
CREATE TABLE ebics_transaction_segments (
    id                 BIGINT PRIMARY KEY,
    owner              VARCHAR(100) NOT NULL,
    ebics_transaction_id BIGINT     NOT NULL,
    segment_number     INTEGER      NOT NULL,
    payload_hash       TEXT         NOT NULL DEFAULT '',
    payload_ref        TEXT         NOT NULL DEFAULT '',
    status             VARCHAR(30)  NOT NULL,
    created_at         DATETIME     NOT NULL,
    updated_at         DATETIME     NOT NULL,
    CONSTRAINT unique_ebics_tx_segment UNIQUE (ebics_transaction_id, segment_number)
);
```

## 4.6 Table `ebics_nonce_window`

```sql
CREATE TABLE ebics_nonce_window (
    id                 BIGINT PRIMARY KEY,
    owner              VARCHAR(100) NOT NULL,
    ebics_host_id      BIGINT       NOT NULL,
    ebics_subscriber_id BIGINT      NOT NULL,
    nonce              VARCHAR(255) NOT NULL,
    seen_at            DATETIME     NOT NULL,
    expires_at         DATETIME     NOT NULL,
    CONSTRAINT unique_ebics_nonce UNIQUE (ebics_host_id, ebics_subscriber_id, nonce)
);
```

## 4.7 Table `ebics_initialization_workflows`

```sql
CREATE TABLE ebics_initialization_workflows (
    id                 BIGINT PRIMARY KEY,
    owner              VARCHAR(100) NOT NULL,
    ebics_subscriber_id BIGINT      NOT NULL,
    init_mode          VARCHAR(20)  NOT NULL,
    status             VARCHAR(40)  NOT NULL,
    ini_transfer_id    BIGINT,
    hia_transfer_id    BIGINT,
    h3k_transfer_id    BIGINT,
    hpb_transfer_id    BIGINT,
    letter_type        VARCHAR(20)  NOT NULL DEFAULT '',
    letter_format      VARCHAR(20)  NOT NULL DEFAULT '',
    letter_storage_ref TEXT         NOT NULL DEFAULT '',
    letter_generated_at DATETIME,
    letter_sent_at     DATETIME,
    activated_at       DATETIME,
    rejected_at        DATETIME,
    operator_name      VARCHAR(100) NOT NULL DEFAULT '',
    evidence           TEXT         NOT NULL DEFAULT '{}',
    created_at         DATETIME     NOT NULL,
    updated_at         DATETIME     NOT NULL
);
```

## 4.8 Table `ebics_key_lifecycles`

```sql
CREATE TABLE ebics_key_lifecycles (
    id                    BIGINT PRIMARY KEY,
    owner                 VARCHAR(100) NOT NULL,
    ebics_subscriber_id   BIGINT,
    ebics_host_id         BIGINT,
    key_usage             VARCHAR(30)  NOT NULL,
    current_credential_id BIGINT,
    next_credential_id    BIGINT,
    rotation_type         VARCHAR(30)  NOT NULL,
    status                VARCHAR(40)  NOT NULL,
    requested_at          DATETIME,
    sent_at               DATETIME,
    activated_at          DATETIME,
    retired_at            DATETIME,
    operator_name         VARCHAR(100) NOT NULL DEFAULT '',
    evidence              TEXT         NOT NULL DEFAULT '{}',
    created_at            DATETIME     NOT NULL,
    updated_at            DATETIME     NOT NULL
);
```

## 4.9 Table `ebics_rtn_events`

```sql
CREATE TABLE ebics_rtn_events (
    id                  BIGINT PRIMARY KEY,
    owner               VARCHAR(100) NOT NULL,
    source              VARCHAR(100) NOT NULL,
    event_id            VARCHAR(255) NOT NULL DEFAULT '',
    correlation_id      VARCHAR(255) NOT NULL DEFAULT '',
    idempotence_key     VARCHAR(255) NOT NULL,
    ebics_host_id       BIGINT,
    ebics_subscriber_id BIGINT,
    profile_id          VARCHAR(100) NOT NULL DEFAULT '',
    order_type_hint     VARCHAR(20)  NOT NULL DEFAULT '',
    payload             TEXT         NOT NULL DEFAULT '{}',
    status              VARCHAR(30)  NOT NULL,
    attempts            INTEGER      NOT NULL DEFAULT 0,
    next_retry_at       DATETIME,
    received_at         DATETIME     NOT NULL,
    processed_at        DATETIME,
    last_error          TEXT         NOT NULL DEFAULT '',
    CONSTRAINT unique_ebics_rtn_idempotence UNIQUE (owner, idempotence_key)
);
```

## 4.10 Table `ebics_event_outbox`

```sql
CREATE TABLE ebics_event_outbox (
    id                  BIGINT PRIMARY KEY,
    owner               VARCHAR(100) NOT NULL,
    aggregate_type      VARCHAR(50)  NOT NULL,
    aggregate_id        VARCHAR(255) NOT NULL,
    event_type          VARCHAR(100) NOT NULL,
    payload             TEXT         NOT NULL DEFAULT '{}',
    status              VARCHAR(30)  NOT NULL,
    attempts            INTEGER      NOT NULL DEFAULT 0,
    next_retry_at       DATETIME,
    published_at        DATETIME,
    last_error          TEXT         NOT NULL DEFAULT '',
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL
);
```

## 4.11 Table `ebics_contract_views`

```sql
CREATE TABLE ebics_contract_views (
    id                  BIGINT PRIMARY KEY,
    owner               VARCHAR(100) NOT NULL,
    ebics_host_id       BIGINT       NOT NULL,
    ebics_subscriber_id BIGINT,
    source_order_type   VARCHAR(20)  NOT NULL,
    source_operation_id BIGINT,
    version_tag         VARCHAR(100) NOT NULL DEFAULT '',
    status              VARCHAR(30)  NOT NULL,
    fetched_at          DATETIME     NOT NULL,
    valid_from          DATETIME,
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL
);
```

## 4.11 bis Table `ebics_contract_view_items`

```sql
CREATE TABLE ebics_contract_view_items (
    id                    BIGINT PRIMARY KEY,
    owner                 VARCHAR(100) NOT NULL,
    contract_view_id      BIGINT       NOT NULL,
    item_type             VARCHAR(30)  NOT NULL,
    item_key              VARCHAR(255) NOT NULL DEFAULT '',
    order_type            VARCHAR(20)  NOT NULL DEFAULT '',
    service_name          VARCHAR(40)  NOT NULL DEFAULT '',
    service_option        VARCHAR(40)  NOT NULL DEFAULT '',
    scope                 VARCHAR(40)  NOT NULL DEFAULT '',
    msg_name              VARCHAR(100) NOT NULL DEFAULT '',
    container_type        VARCHAR(40)  NOT NULL DEFAULT '',
    admin_order_type      VARCHAR(20)  NOT NULL DEFAULT '',
    authorisation_level   VARCHAR(10)  NOT NULL DEFAULT '',
    account_id            VARCHAR(64)  NOT NULL DEFAULT '',
    max_amount_value      DECIMAL(24,4),
    max_amount_currency   VARCHAR(3)   NOT NULL DEFAULT '',
    is_enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    payload               TEXT         NOT NULL DEFAULT '{}',
    created_at            DATETIME     NOT NULL,
    updated_at            DATETIME     NOT NULL
);
```

## 4.12 Table `business_connector_deliveries`

```sql
CREATE TABLE business_connector_deliveries (
    id                  BIGINT PRIMARY KEY,
    owner               VARCHAR(100) NOT NULL,
    connector_type      VARCHAR(30)  NOT NULL,
    connector_profile   VARCHAR(100) NOT NULL,
    aggregate_type      VARCHAR(50)  NOT NULL,
    aggregate_id        VARCHAR(255) NOT NULL,
    event_type          VARCHAR(100) NOT NULL,
    correlation_id      VARCHAR(255) NOT NULL DEFAULT '',
    idempotence_key     VARCHAR(255) NOT NULL DEFAULT '',
    status              VARCHAR(30)  NOT NULL,
    attempts            INTEGER      NOT NULL DEFAULT 0,
    next_retry_at       DATETIME,
    delivered_at        DATETIME,
    last_error          TEXT         NOT NULL DEFAULT '',
    payload             TEXT         NOT NULL DEFAULT '{}',
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL
);
```

## 5. Surfaces REST a ajouter

Principe:

- prolonger les ressources existantes quand cela a du sens;
- creer de nouvelles ressources dediees pour les workflows et evenements
  vraiment nouveaux.

## 5.1 Extensions sur ressources existantes

### Serveurs

- `GET /api/servers/{server}`:
  enrichir avec section `ebics` si protocole `ebics`
- `PATCH /api/servers/{server}`:
  mise a jour du `protoConfig` EBICS
- `GET /api/servers/{server}/credentials`:
  reutiliser pour les credentials serveur EBICS

### Clients

- `GET /api/clients/{client}`
- `PATCH /api/clients/{client}`

### Partenaires

- `GET /api/partners/{partner}`
- `PATCH /api/partners/{partner}`

### Comptes

- `GET /api/servers/{server}/accounts/{login}`
- `GET /api/partners/{partner}/accounts/{login}`

## 5.2 Nouvelles ressources EBICS

### Hosts

- `GET /api/ebics/hosts`
- `POST /api/ebics/hosts`
- `GET /api/ebics/hosts/{host}`
- `PATCH /api/ebics/hosts/{host}`
- `DELETE /api/ebics/hosts/{host}`

### Subscribers

- `GET /api/ebics/subscribers`
- `POST /api/ebics/subscribers`
- `GET /api/ebics/subscribers/{subscriber}`
- `PATCH /api/ebics/subscribers/{subscriber}`
- `DELETE /api/ebics/subscribers/{subscriber}`

### Initialisation workflows

- `POST /api/ebics/subscribers/{subscriber}/initializations`
- `GET /api/ebics/initializations`
- `GET /api/ebics/initializations/{workflow}`
- `POST /api/ebics/initializations/{workflow}/generate-letter`
- `GET /api/ebics/initializations/{workflow}/letter`
- `POST /api/ebics/initializations/{workflow}/mark-letter-sent`
- `POST /api/ebics/initializations/{workflow}/confirm-external-activation`
- `POST /api/ebics/initializations/{workflow}/cancel`

### Key rotations

- `POST /api/ebics/subscribers/{subscriber}/key-rotations`
- `GET /api/ebics/key-rotations`
- `GET /api/ebics/key-rotations/{rotation}`
- `POST /api/ebics/key-rotations/{rotation}/prepare`
- `POST /api/ebics/key-rotations/{rotation}/send`
- `POST /api/ebics/key-rotations/{rotation}/activate`
- `POST /api/ebics/key-rotations/{rotation}/retire`
- `POST /api/ebics/key-rotations/{rotation}/cancel`

### Bank keys

- `GET /api/ebics/hosts/{host}/bank-keys`
- `POST /api/ebics/hosts/{host}/bank-keys/refresh-hpb`
- `POST /api/ebics/hosts/{host}/bank-keys/confirm`

### Contract views

- `POST /api/ebics/partners/{partner}/contract-view/refresh`
- `GET /api/ebics/partners/{partner}/contract-view`
- `GET /api/ebics/partners/{partner}/contract-view/capabilities`
- `GET /api/ebics/partners/{partner}/contract-view/permissions`

### Transactions and order history

- `GET /api/ebics/transactions`
- `GET /api/ebics/transactions/{tx}`
- `GET /api/ebics/orders`
- `GET /api/ebics/orders/{id}`
- `GET /api/ebics/signatures`
- `GET /api/ebics/signatures/{id}`

### RTN

- `GET /api/ebics/rtn/events`
- `GET /api/ebics/rtn/events/{event}`
- `POST /api/ebics/rtn/events/{event}/retry`
- `POST /api/ebics/rtn/events/{event}/quarantine`
- `POST /api/ebics/rtn/providers`
- `GET /api/ebics/rtn/providers`
- `PATCH /api/ebics/rtn/providers/{provider}`

### Passe-plat metier

- `POST /api/ebics/events/outbox/replay`
- `GET /api/ebics/events/outbox`
- `GET /api/ebics/events/outbox/{event}`

### Connecteurs metier

- `GET /api/business-connectors`
- `POST /api/business-connectors`
- `GET /api/business-connectors/{connector}`
- `PATCH /api/business-connectors/{connector}`
- `GET /api/business-connectors/{connector}/deliveries`
- `POST /api/business-connectors/{connector}/deliveries/{delivery}/retry`

## 6. Surfaces CLI a ajouter

Principe:

- rester aligne avec la grammaire actuelle `waarp-gateway <resource> <action>`;
- eviter une CLI totalement separee.

## 6.1 Commandes proposees

### Hosts

```text
waarp-gateway ebics host add
waarp-gateway ebics host get
waarp-gateway ebics host list
waarp-gateway ebics host update
waarp-gateway ebics host delete
```

### Subscribers

```text
waarp-gateway ebics subscriber add
waarp-gateway ebics subscriber get
waarp-gateway ebics subscriber list
waarp-gateway ebics subscriber update
waarp-gateway ebics subscriber delete
```

### Initialisation

```text
waarp-gateway ebics init start
waarp-gateway ebics init get
waarp-gateway ebics init list
waarp-gateway ebics init generate-letter
waarp-gateway ebics init download-letter
waarp-gateway ebics init mark-letter-sent
waarp-gateway ebics init confirm-external-activation
waarp-gateway ebics init cancel
```

### Rotation de cles

```text
waarp-gateway ebics key-rotation start
waarp-gateway ebics key-rotation get
waarp-gateway ebics key-rotation list
waarp-gateway ebics key-rotation prepare
waarp-gateway ebics key-rotation send
waarp-gateway ebics key-rotation activate
waarp-gateway ebics key-rotation retire
waarp-gateway ebics key-rotation cancel
```

### Bank keys and admin orders

```text
waarp-gateway ebics bank-keys refresh-hpb
waarp-gateway ebics bank-keys get
waarp-gateway ebics order hev
waarp-gateway ebics order hpb
waarp-gateway ebics order hpd
waarp-gateway ebics order htd
waarp-gateway ebics order hkd
waarp-gateway ebics order haa
```

### Contract view

```text
waarp-gateway ebics contract-view refresh
waarp-gateway ebics contract-view get
waarp-gateway ebics contract-view capabilities
waarp-gateway ebics contract-view permissions
```

### RTN

```text
waarp-gateway ebics rtn provider add
waarp-gateway ebics rtn provider get
waarp-gateway ebics rtn provider list
waarp-gateway ebics rtn provider update
waarp-gateway ebics rtn event get
waarp-gateway ebics rtn event list
waarp-gateway ebics rtn event retry
waarp-gateway ebics rtn event quarantine
```

### Passe-plat metier

```text
waarp-gateway ebics event-outbox list
waarp-gateway ebics event-outbox get
waarp-gateway ebics event-outbox replay
```

### Connecteurs metier

```text
waarp-gateway business-connector add
waarp-gateway business-connector get
waarp-gateway business-connector list
waarp-gateway business-connector update
waarp-gateway business-connector delivery list
waarp-gateway business-connector delivery retry
```

## 7. Matrice de mapping Gateway <-> EBICS

| Concept Gateway | Concept EBICS | Strategie |
| --- | --- | --- |
| `LocalAgent` | Serveur EBICS local | Reutilisation directe |
| `Client` | Client EBICS local | Reutilisation directe |
| `RemoteAgent` | Banque / host distant | Reutilisation + lien `ebics_hosts` |
| `LocalAccount` | Utilisateur EBICS local | Reutilisation + lien `ebics_subscribers` |
| `RemoteAccount` | Utilisateur EBICS distant | Reutilisation + lien `ebics_subscribers` |
| `Credential` | Cle/certificat/secret EBICS | Reutilisation avec types dedies |
| `Rule` | Habilitation d'usage | Reutilisation partielle |
| `Transfer` | Echange fichier EBICS | Reutilisation seulement pour ordres fichier |
| `TransferInfo` | Metadonnees d'execution | Reutilisation minimale |
| Historique de transfert | Historique des flux fichier EBICS | Reutilisation |
| Aucun objet direct | `HostID` | Nouvelle table `ebics_hosts` |
| Aucun objet direct | `(PartnerID, UserID)` | Nouvelle table `ebics_subscribers` |
| Aucun objet direct | Bank public keys / digests | Nouvelle table `ebics_bank_keys` |
| Aucun objet direct | Transaction EBICS | Nouvelle table `ebics_transactions` |
| Aucun objet direct | Segment EBICS | Nouvelle table `ebics_transaction_segments` |
| Aucun objet direct | Nonce anti-rejeu | Nouvelle table `ebics_nonce_window` |
| Aucun objet direct | Suivi technique d'initialisation | Nouvelle table `ebics_initialization_workflows` |
| Aucun objet direct | Rotation de cles | Nouvelle table `ebics_key_lifecycles` |
| Aucun objet direct | Evenement RTN | Nouvelle table `ebics_rtn_events` |
| Aucun objet direct | Evenement technique sortant | Nouvelle table `ebics_event_outbox` |
| Aucun objet direct | Vue contractuelle technique | Nouvelle table `ebics_contract_views` |
| Aucun objet direct | Livraison connecteur metier | Nouvelle table `business_connector_deliveries` |

## 8. Mapping des ordres EBICS

| Ordre | Type | Support cible | Mode d'integration |
| --- | --- | --- | --- |
| `HEV` | Administratif | P1 | Historique EBICS dedie |
| `INI` | Initialisation | P1 | Workflow d'initialisation |
| `HIA` | Initialisation | P1 | Workflow d'initialisation |
| `H3K` | Initialisation/rotation | P2 | Workflow d'initialisation/rotation |
| `HPB` | Admin/Trust bootstrap | P1 | Historique EBICS + bank keys |
| `PUB` | Rotation | P2 | Workflow de rotation |
| `HCA` | Rotation | P2 | Workflow de rotation |
| `HCS` | Rotation | P2 | Workflow de rotation |
| `HSA` | Rotation | P2 | Workflow de rotation |
| `SPR` | Rotation/revocation | P2 | Workflow de rotation |
| `HPD` | Administratif | P2 | Historique EBICS dedie |
| `HTD` | Administratif | P2 | Historique EBICS dedie |
| `HKD` | Administratif | P2 | Historique EBICS dedie |
| `HAA` | Administratif / RTN bootstrap | P2 | Historique EBICS dedie / RTN |
| `FUL` | Fichier upload | P1 | `Transfer` + `Pipeline` |
| `FDL` | Fichier download | P1 | `Transfer` + `Pipeline` |
| `BTU` | Fichier upload | P1 | `Transfer` + `Pipeline` |
| `BTD` | Fichier download | P1 | `Transfer` + `Pipeline` |
| `HAC` | Reporting / pull | P3 | Historique EBICS ou RTN auto-pull |
| `HVD` | Reporting | P3 | Historique EBICS dedie |
| `HVE` | Reporting | P3 | Historique EBICS dedie |
| `HVT` | Reporting | P3 | Historique EBICS dedie |
| `HVU` | Reporting/signature | P3 | Historique EBICS dedie |
| `HVZ` | Reporting/signature | P3 | Historique EBICS dedie |
| `HVS` | Signature/cancel | P3 | Historique EBICS dedie + passe-plat metier |

## 9. Priorisation de code par repertoire

Ordre recommande de developpement:

1. `pkg/protocols/modules/ebics/`
2. `pkg/model/` pour les nouveaux modeles EBICS
3. `pkg/database/migrations/`
4. `pkg/admin/rest/` et `pkg/admin/rest/api/`
5. `pkg/cmd/client/`
6. `pkg/admin/gui/` et `pkg/admin/gui/v2/`
7. `doc/source/reference/`

## 10. Jeux de tests a prevoir

### Unitaires

- validation des `ProtoConfig`;
- stores SQL EBICS;
- parsers de vue contractuelle `HPD` / `HKD` / `HTD` / `HAA`;
- generation de lettre;
- transitions d'etats techniques d'initialisation;
- transitions de workflow rotation;
- deduplication RTN;
- publication outbox vers connecteur AMQP.

### Integration

- creation serveur/client/partenaire EBICS via REST;
- onboarding technique avec confirmation externe;
- collecte `HPD` / `HTD` / `HKD` / `HAA` et stockage de la vue contractuelle;
- flux `FUL` / `FDL` / `BTU` / `BTD`;
- refresh `HPB`;
- rotation de cles;
- RTN -> auto-pull -> historique;
- emission d'un evenement technique vers l'application metier.
- emission d'un evenement technique via AMQP.

### Non-regression

- demarrage Gateway sans ressource EBICS;
- coexistence EBICS avec protocoles existants;
- maintien de l'historique et des tasks standards.

## 11. Risques de backlog

- vouloir traiter RTN trop tot avant d'avoir les stores stables;
- trop charger `TransferInfo` au lieu d'assumer les nouvelles tables;
- oublier la purge et la retention des nonces et evenements RTN;
- ne pas distinguer succes technique et activation metier;
- glisser vers un workflow bancaire metier dans Gateway;
- transformer la vue contractuelle technique en pseudo-contrat metier;
- sur-specialiser EBICS alors que le connecteur messaging doit rester
  transverse a Gateway;
- sous-estimer les surfaces REST/CLI a maintenir.

## 12. Definition of done par grand axe

### Initialisation

- suivi technique trace de bout en bout;
- lettre generable et telechargeable;
- activation impossible sans confirmation externe explicite.

### Flux fichier

- ordre EBICS visible dans l'historique Gateway;
- relation `TransactionID` / `Transfer.ID` consultable;
- tasks executees.

### Rotation

- coexistence ancien/nouveau materiel supportee;
- bascule controlee;
- preuves consultees.

### RTN

- message dedupplique;
- plan de pull resolu;
- execution et historique relies a l'evenement.

### Passe-plat metier

- evenement technique sortant produit;
- rejeu possible en cas d'echec de livraison;
- aucune decision metier embarquee dans Gateway.

### Contrat technique

- vue `HPD` / `HKD` / `HTD` / `HAA` rafraichissable;
- ordres manifestement hors contrat bloquables avant emission;
- aucune substitution au contrat juridique ou au workflow metier.

### Connecteurs messaging

- publication fiable vers `AMQP 0.9.1` ou `AMQP 1.0`;
- correlation et rejeu exploitables;
- aucune dependance a un broker ou a un format metier unique.
