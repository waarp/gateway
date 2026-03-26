# Modele SQL detaille de reference pour EBICS

## 1. Objet

Ce document derive les specs et la matrice des ordres vers un modele SQL
detaille de reference, exploitable pour les futures migrations.

Le but n'est pas d'imposer une syntaxe parfaite pour un SGBD donne, mais de
figer:

- les tables;
- les cles;
- les unicites;
- les relations;
- les index utiles;
- les responsabilites de chaque objet.

Important:

- les blocs `CREATE TABLE` ci-dessous doivent etre lus comme un pseudo-DDL de
  reference logique;
- ils ne constituent pas encore les migrations physiques finales multi-SGBD;
- la traduction finale doit etre portee par l'ORM existant et, si necessaire,
  par les mecanismes de migration deja utilises par Gateway.

## 1.1 Contrainte transversale multi-SGBD

Gateway etant multi-SGBD, ce modele doit rester compatible avec l'ORM existant
et ne pas deriver vers un design dependent d'un moteur SQL particulier.

Cela implique:

- privilegier des colonnes simples, bien mappables via XORM;
- eviter les types vendor-specifiques comme `JSONB`, `ARRAY`, enums natifs ou
  autres variantes proprietaires;
- preferer des tables filles structurees aux gros blobs JSON lorsqu'une UI, une
  API ou une validation doivent exploiter finement les donnees;
- traiter les champs `TEXT` contenant du JSON comme une exception justifiee,
  pas comme une base de modelisation.

Consequence importante:

- des elements de pseudo-DDL comme `AUTO_INCREMENT`, `BOOLEAN`, `CHECK` ou
  certains `DEFAULT` textuels ne doivent pas etre lus comme des choix physiques
  definitifs;
- ils expriment une intention de modelisation, qui devra ensuite etre projetee
  proprement dans l'outillage Gateway compatible MySQL/PostgreSQL/SQLite ou
  autres SGBD supportes.

## 2. Principes de modelisation

- `EbicsOperation` est le pivot operationnel;
- `ebics_transactions` porte l'etat transactionnel protocolaire;
- `ebics_transaction_segments` porte l'etat fin de segmentation/recovery;
- `Transfer` reste reserve aux flux fichier;
- les identifiants Gateway et EBICS coexistent sans fusion;
- `TransferInfo` ne porte pas les relations structurelles principales.

## 3. Table `ebics_hosts`

```sql
CREATE TABLE ebics_hosts (
    id                  BIGINT       NOT NULL AUTO_INCREMENT,
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

    CONSTRAINT ebics_hosts_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_host_name UNIQUE (owner, name),
    CONSTRAINT unique_ebics_host_id UNIQUE (owner, host_id),
    CONSTRAINT ebics_host_mode_check CHECK (mode IN ('server', 'remote'))
);
```

Indexes recommandes:

- `(owner, gateway_entity_type, gateway_entity_id)`
- `(owner, status)`

## 4. Table `ebics_subscribers`

```sql
CREATE TABLE ebics_subscribers (
    id                BIGINT       NOT NULL AUTO_INCREMENT,
    owner             VARCHAR(100) NOT NULL,
    ebics_host_id     BIGINT       NOT NULL,
    partner_id        VARCHAR(100) NOT NULL,
    user_id           VARCHAR(100) NOT NULL,
    account_role      VARCHAR(30)  NOT NULL,
    local_account_id  BIGINT,
    remote_account_id BIGINT,
    state             VARCHAR(30)  NOT NULL,
    permissions       TEXT         NOT NULL DEFAULT '{}',
    options           TEXT         NOT NULL DEFAULT '{}',
    created_at        DATETIME     NOT NULL,
    updated_at        DATETIME     NOT NULL,

    CONSTRAINT ebics_subscribers_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_subscriber UNIQUE (owner, ebics_host_id, partner_id, user_id),
    CONSTRAINT ebics_subscriber_host_fkey FOREIGN KEY (ebics_host_id)
        REFERENCES ebics_hosts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT ebics_subscriber_local_account_fkey FOREIGN KEY (local_account_id)
        REFERENCES local_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_subscriber_remote_account_fkey FOREIGN KEY (remote_account_id)
        REFERENCES remote_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_subscriber_owner_check CHECK (
        CASE WHEN local_account_id  IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN remote_account_id IS NOT NULL THEN 1 ELSE 0 END = 1
    )
);
```

Indexes recommandes:

- `(owner, partner_id, user_id)`
- `(owner, state)`

## 5. Table `ebics_bank_keys`

```sql
CREATE TABLE ebics_bank_keys (
    id               BIGINT       NOT NULL AUTO_INCREMENT,
    owner            VARCHAR(100) NOT NULL,
    ebics_host_id    BIGINT       NOT NULL,
    key_usage        VARCHAR(30)  NOT NULL,
    key_version      VARCHAR(30),
    public_key_pem   TEXT         NOT NULL,
    digest_algorithm VARCHAR(30)  NOT NULL,
    digest_value     VARCHAR(255) NOT NULL,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    valid_from       DATETIME,
    valid_to         DATETIME,
    created_at       DATETIME     NOT NULL,
    updated_at       DATETIME     NOT NULL,

    CONSTRAINT ebics_bank_keys_pkey PRIMARY KEY (id),
    CONSTRAINT ebics_bank_keys_host_fkey FOREIGN KEY (ebics_host_id)
        REFERENCES ebics_hosts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT unique_ebics_bank_key UNIQUE (owner, ebics_host_id, key_usage, digest_value)
);
```

Indexes recommandes:

- `(owner, ebics_host_id, key_usage, is_active)`

## 6. Table `ebics_contract_views`

```sql
CREATE TABLE ebics_contract_views (
    id                  BIGINT       NOT NULL AUTO_INCREMENT,
    owner               VARCHAR(100) NOT NULL,
    ebics_host_id       BIGINT       NOT NULL,
    ebics_subscriber_id BIGINT,
    source_order_type   VARCHAR(10)  NOT NULL,
    source_operation_id BIGINT,
    version_tag         VARCHAR(100),
    status              VARCHAR(30)  NOT NULL,
    fetched_at          DATETIME     NOT NULL,
    valid_from          DATETIME,
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL,

    CONSTRAINT ebics_contract_views_pkey PRIMARY KEY (id),
    CONSTRAINT ebics_contract_views_host_fkey FOREIGN KEY (ebics_host_id)
        REFERENCES ebics_hosts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT ebics_contract_views_subscriber_fkey FOREIGN KEY (ebics_subscriber_id)
        REFERENCES ebics_subscribers (id) ON UPDATE RESTRICT ON DELETE SET NULL
);
```

Indexes recommandes:

- `(owner, ebics_host_id, fetched_at DESC)`
- `(owner, ebics_subscriber_id, fetched_at DESC)`
- `(owner, status)`

## 6 bis. Table `ebics_contract_view_items`

```sql
CREATE TABLE ebics_contract_view_items (
    id                    BIGINT       NOT NULL AUTO_INCREMENT,
    owner                 VARCHAR(100) NOT NULL,
    contract_view_id      BIGINT       NOT NULL,
    item_type             VARCHAR(30)  NOT NULL,
    item_key              VARCHAR(255) NOT NULL DEFAULT '',
    order_type            VARCHAR(10)  NOT NULL DEFAULT '',
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
    updated_at            DATETIME     NOT NULL,

    CONSTRAINT ebics_contract_view_items_pkey PRIMARY KEY (id),
    CONSTRAINT ebics_contract_view_items_view_fkey FOREIGN KEY (contract_view_id)
        REFERENCES ebics_contract_views (id) ON UPDATE RESTRICT ON DELETE CASCADE
);
```

Indexes recommandes:

- `(owner, contract_view_id, item_type)`
- `(owner, item_type, order_type)`
- `(owner, service_name, service_option, scope, msg_name)`
- `(owner, account_id)`

## 7. Table `ebics_operations`

```sql
CREATE TABLE ebics_operations (
    id                 BIGINT       NOT NULL AUTO_INCREMENT,
    owner              VARCHAR(100) NOT NULL,
    local_agent_id     BIGINT,
    client_id          BIGINT,
    remote_agent_id    BIGINT,
    local_account_id   BIGINT,
    remote_account_id  BIGINT,
    ebics_host_id      BIGINT,
    ebics_subscriber_id BIGINT,
    operation_type     VARCHAR(30)  NOT NULL,
    order_type         VARCHAR(10)  NOT NULL,
    direction          VARCHAR(20)  NOT NULL,
    transport_mode     VARCHAR(20)  NOT NULL,
    transaction_id     VARCHAR(100),
    request_id         VARCHAR(100),
    correlation_id     VARCHAR(100),
    ebics_version      VARCHAR(10),
    status             VARCHAR(40)  NOT NULL,
    severity           VARCHAR(20)  NOT NULL,
    technical_return_code    VARCHAR(20),
    technical_return_message TEXT         NOT NULL DEFAULT '',
    business_return_code     VARCHAR(20),
    business_return_message  TEXT         NOT NULL DEFAULT '',
    gateway_outcome          VARCHAR(40)  NOT NULL DEFAULT '',
    retry_decision           VARCHAR(40)  NOT NULL DEFAULT '',
    manual_action_required   BOOLEAN      NOT NULL DEFAULT FALSE,
    transfer_id        BIGINT,
    contract_view_id   BIGINT,
    rtn_event_id       BIGINT,
    metadata           TEXT         NOT NULL DEFAULT '{}',
    started_at         DATETIME,
    finished_at        DATETIME,
    created_at         DATETIME     NOT NULL,
    updated_at         DATETIME     NOT NULL,

    CONSTRAINT ebics_operations_pkey PRIMARY KEY (id),
    CONSTRAINT ebics_operations_local_agent_fkey FOREIGN KEY (local_agent_id)
        REFERENCES local_agents (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_client_fkey FOREIGN KEY (client_id)
        REFERENCES clients (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_remote_agent_fkey FOREIGN KEY (remote_agent_id)
        REFERENCES remote_agents (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_local_account_fkey FOREIGN KEY (local_account_id)
        REFERENCES local_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_remote_account_fkey FOREIGN KEY (remote_account_id)
        REFERENCES remote_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_host_fkey FOREIGN KEY (ebics_host_id)
        REFERENCES ebics_hosts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_subscriber_fkey FOREIGN KEY (ebics_subscriber_id)
        REFERENCES ebics_subscribers (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_operations_transfer_fkey FOREIGN KEY (transfer_id)
        REFERENCES transfers (id) ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT ebics_operations_contract_view_fkey FOREIGN KEY (contract_view_id)
        REFERENCES ebics_contract_views (id) ON UPDATE RESTRICT ON DELETE SET NULL
);
```

Indexes recommandes:

- `(owner, status, created_at DESC)`
- `(owner, order_type, created_at DESC)`
- `(owner, operation_type, created_at DESC)`
- `(owner, correlation_id)`
- `(owner, transaction_id)`
- `(owner, request_id)`
- `(owner, transfer_id)`
- `(owner, ebics_subscriber_id, created_at DESC)`

Unicites recommandees:

- pas d'unicite globale forte sur `transaction_id` seule;
- unicite defensive possible sur `(owner, ebics_subscriber_id, request_id)`
  quand `request_id` est renseigne.

## 8. Table `ebics_transactions`

```sql
CREATE TABLE ebics_transactions (
    id                  BIGINT       NOT NULL AUTO_INCREMENT,
    owner               VARCHAR(100) NOT NULL,
    ebics_operation_id  BIGINT,
    ebics_host_id       BIGINT       NOT NULL,
    ebics_subscriber_id BIGINT       NOT NULL,
    transaction_id      VARCHAR(100) NOT NULL,
    order_type          VARCHAR(10)  NOT NULL,
    transfer_id         BIGINT,
    status              VARCHAR(40)  NOT NULL,
    direction           VARCHAR(20)  NOT NULL,
    segment_count       INTEGER      NOT NULL DEFAULT 0,
    current_segment     INTEGER      NOT NULL DEFAULT 0,
    total_size          BIGINT       NOT NULL DEFAULT -1,
    resumed_from_tx_id  BIGINT,
    metadata            TEXT         NOT NULL DEFAULT '{}',
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL,

    CONSTRAINT ebics_transactions_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_transaction UNIQUE (owner, ebics_subscriber_id, transaction_id),
    CONSTRAINT ebics_transactions_operation_fkey FOREIGN KEY (ebics_operation_id)
        REFERENCES ebics_operations (id) ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT ebics_transactions_host_fkey FOREIGN KEY (ebics_host_id)
        REFERENCES ebics_hosts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_transactions_subscriber_fkey FOREIGN KEY (ebics_subscriber_id)
        REFERENCES ebics_subscribers (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT ebics_transactions_transfer_fkey FOREIGN KEY (transfer_id)
        REFERENCES transfers (id) ON UPDATE RESTRICT ON DELETE SET NULL,
    CONSTRAINT ebics_transactions_resumed_from_fkey FOREIGN KEY (resumed_from_tx_id)
        REFERENCES ebics_transactions (id) ON UPDATE RESTRICT ON DELETE SET NULL
);
```

Indexes recommandes:

- `(owner, status, created_at DESC)`
- `(owner, order_type, created_at DESC)`
- `(owner, transfer_id)`
- `(owner, ebics_operation_id)`

## 9. Table `ebics_transaction_segments`

```sql
CREATE TABLE ebics_transaction_segments (
    id                   BIGINT       NOT NULL AUTO_INCREMENT,
    owner                VARCHAR(100) NOT NULL,
    ebics_transaction_id BIGINT       NOT NULL,
    segment_number       INTEGER      NOT NULL,
    segment_status       VARCHAR(30)  NOT NULL,
    payload_size         BIGINT       NOT NULL DEFAULT 0,
    checksum             VARCHAR(255),
    stored_payload_ref   VARCHAR(255),
    metadata             TEXT         NOT NULL DEFAULT '{}',
    created_at           DATETIME     NOT NULL,
    updated_at           DATETIME     NOT NULL,

    CONSTRAINT ebics_transaction_segments_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_tx_segment UNIQUE (ebics_transaction_id, segment_number),
    CONSTRAINT ebics_tx_segments_tx_fkey FOREIGN KEY (ebics_transaction_id)
        REFERENCES ebics_transactions (id) ON UPDATE RESTRICT ON DELETE CASCADE
);
```

Indexes recommandes:

- `(owner, segment_status)`
- `(owner, ebics_transaction_id, segment_number)`

## 10. Table `ebics_nonce_window`

```sql
CREATE TABLE ebics_nonce_window (
    id                  BIGINT       NOT NULL AUTO_INCREMENT,
    owner               VARCHAR(100) NOT NULL,
    ebics_host_id       BIGINT       NOT NULL,
    ebics_subscriber_id BIGINT,
    nonce_value         VARCHAR(255) NOT NULL,
    request_timestamp   DATETIME     NOT NULL,
    expires_at          DATETIME     NOT NULL,
    created_at          DATETIME     NOT NULL,

    CONSTRAINT ebics_nonce_window_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_nonce UNIQUE (owner, ebics_host_id, nonce_value),
    CONSTRAINT ebics_nonce_host_fkey FOREIGN KEY (ebics_host_id)
        REFERENCES ebics_hosts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT ebics_nonce_subscriber_fkey FOREIGN KEY (ebics_subscriber_id)
        REFERENCES ebics_subscribers (id) ON UPDATE RESTRICT ON DELETE SET NULL
);
```

Indexes recommandes:

- `(owner, expires_at)`

## 11. Table `ebics_initialization_workflows`

```sql
CREATE TABLE ebics_initialization_workflows (
    id                  BIGINT       NOT NULL AUTO_INCREMENT,
    owner               VARCHAR(100) NOT NULL,
    ebics_subscriber_id BIGINT       NOT NULL,
    status              VARCHAR(40)  NOT NULL,
    init_mode           VARCHAR(20)  NOT NULL,
    ini_operation_id    BIGINT,
    hia_operation_id    BIGINT,
    h3k_operation_id    BIGINT,
    hpb_operation_id    BIGINT,
    letter_type         VARCHAR(20),
    letter_generated_at DATETIME,
    letter_delivery_mode VARCHAR(20),
    letter_sent_at      DATETIME,
    bank_feedback       TEXT         NOT NULL DEFAULT '',
    activated_at        DATETIME,
    operator_name       VARCHAR(100),
    evidence            TEXT         NOT NULL DEFAULT '{}',
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL,

    CONSTRAINT ebics_init_workflows_pkey PRIMARY KEY (id),
    CONSTRAINT ebics_init_workflows_subscriber_fkey FOREIGN KEY (ebics_subscriber_id)
        REFERENCES ebics_subscribers (id) ON UPDATE RESTRICT ON DELETE CASCADE
);
```

## 12. Table `ebics_key_lifecycles`

```sql
CREATE TABLE ebics_key_lifecycles (
    id                    BIGINT       NOT NULL AUTO_INCREMENT,
    owner                 VARCHAR(100) NOT NULL,
    ebics_subscriber_id   BIGINT       NOT NULL,
    key_usage             VARCHAR(30)  NOT NULL,
    current_credential_id BIGINT,
    next_credential_id    BIGINT,
    rotation_type         VARCHAR(30)  NOT NULL,
    status                VARCHAR(40)  NOT NULL,
    trigger_operation_id  BIGINT,
    requested_at          DATETIME,
    sent_at               DATETIME,
    activated_at          DATETIME,
    retired_at            DATETIME,
    operator_name         VARCHAR(100),
    evidence              TEXT         NOT NULL DEFAULT '{}',
    created_at            DATETIME     NOT NULL,
    updated_at            DATETIME     NOT NULL,

    CONSTRAINT ebics_key_lifecycles_pkey PRIMARY KEY (id),
    CONSTRAINT ebics_key_lifecycles_subscriber_fkey FOREIGN KEY (ebics_subscriber_id)
        REFERENCES ebics_subscribers (id) ON UPDATE RESTRICT ON DELETE CASCADE
);
```

## 13. Table `ebics_rtn_events`

```sql
CREATE TABLE ebics_rtn_events (
    id                  BIGINT       NOT NULL AUTO_INCREMENT,
    owner               VARCHAR(100) NOT NULL,
    source              VARCHAR(50)  NOT NULL,
    event_id            VARCHAR(100) NOT NULL,
    correlation_id      VARCHAR(100),
    idempotence_key     VARCHAR(150) NOT NULL,
    ebics_host_id       BIGINT,
    ebics_subscriber_id BIGINT,
    order_type_hint     VARCHAR(10),
    profile_id          VARCHAR(100),
    payload             TEXT         NOT NULL DEFAULT '{}',
    status              VARCHAR(40)  NOT NULL,
    attempts            INTEGER      NOT NULL DEFAULT 0,
    next_retry_at       DATETIME,
    received_at         DATETIME     NOT NULL,
    processed_at        DATETIME,
    created_at          DATETIME     NOT NULL,
    updated_at          DATETIME     NOT NULL,

    CONSTRAINT ebics_rtn_events_pkey PRIMARY KEY (id),
    CONSTRAINT unique_ebics_rtn_idempotence UNIQUE (owner, idempotence_key)
);
```

Indexes recommandes:

- `(owner, status, received_at DESC)`
- `(owner, correlation_id)`
- `(owner, event_id)`

## 13 bis. Table `ebics_payload_profiles`

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

## 14. Relations structurantes

Relations de reference:

- `ebics_operations.transfer_id -> transfers.id`
- `ebics_transactions.ebics_operation_id -> ebics_operations.id`
- `ebics_transaction_segments.ebics_transaction_id -> ebics_transactions.id`
- `ebics_contract_views.source_operation_id -> ebics_operations.id`
- `ebics_initialization_workflows.*_operation_id -> ebics_operations.id`
- `ebics_key_lifecycles.trigger_operation_id -> ebics_operations.id`
- `ebics_rtn_events` peut etre relie indirectement via `rtn_event_id` dans
  `ebics_operations`

## 15. Decisions SQL importantes

- `ebics_operations` est le pivot administratif et d'exploitation;
- `ebics_transactions` est le pivot de reprise protocolaire;
- les segments ne sont jamais modeles dans `Transfer`;
- `Transfer` reste optionnel dans la chaine EBICS;
- `request_id` et `correlation_id` doivent etre indexables pour la supervision
  et le diagnostic;
- pas d'unicite trop aggressive sur des identifiants dont la granularite doit
  encore etre confirmee avec la librairie et les banques.

## 16. Prochaine derivation

Ce modele SQL detaille doit ensuite etre derive vers:

- les structs `model` cibles;
- les migrations;
- les DTO REST;
- les repositories/stores EBICS.
