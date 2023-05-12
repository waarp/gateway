# Current database schema

**Note**: The following ``CREATE TABLE`` are only there to document the current
database schema. Consequently, they CANNOT be used to initialize a database as
they are written in pseudo-SQL, which most likely will not work on any database
system.

### Table ``version``

```sqlite
CREATE TABLE version (
    current TEXT NOT NULL
)
```

### Table ``users``

```sqlite
CREATE TABLE users (
    id            BIGINT       NOT NULL AUTOINCREMENT,
    owner         VARCHAR(100) NOT NULL,
    username      VARCHAR(100) NOT NULL,
    password_hash TEXT         NOT NULL DEFAULT '',
    permissions   INTEGER      NOT NULL DEFAULT 0,

    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT unique_user UNIQUE (owner, username)
)
```

### Table ``local_agents``

```sqlite
CREATE TABLE local_agents (
    id              BIGINT       NOT NULL AUTOINCREMENT,
    owner           VARCHAR(100) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    protocol        VARCHAR(50)  NOT NULL,
    address         VARCHAR(255) NOT NULL,
    proto_config    TEXT         NOT NULL DEFAULT '{}',
    enabled         BOOLEAN      NOT NULL DEFAULT true,
    root_dir        TEXT         NOT NULL DEFAULT '',
    receive_dir     TEXT         NOT NULL DEFAULT '',
    send_dir        TEXT         NOT NULL DEFAULT '',
    tmp_receive_dir TEXT         NOT NULL DEFAULT '',
    
    CONSTRAINT local_agents_pkey PRIMARY KEY (id),
    CONSTRAINT unique_local_agent UNIQUE (owner, name)
)
```

### Table ``remote_agents``

```sqlite
CREATE TABLE remote_agents (
    id           BIGINT       NOT NULL AUTOINCREMENT,
    name         VARCHAR(100) NOT NULL,
    protocol     VARCHAR(50)  NOT NULL,
    address      VARCHAR(255) NOT NULL,
    proto_config TEXT         NOT NULL DEFAULT '{}',
    
    CONSTRAINT remote_agents_pkey  PRIMARY KEY (id),
    CONSTRAINT unique_remote_agent UNIQUE (name)
)
```

### Table ``local_accounts``

```sqlite
CREATE TABLE local_accounts (
    id             BIGINT       NOT NULL AUTOINCREMENT,
    local_agent_id BIGINT       NOT NULL,
    login          VARCHAR(100) NOT NULL,
    password_hash  TEXT         NOT NULL DEFAULT '',
  
    CONSTRAINT local_accounts_pkey PRIMARY KEY (id),
    CONSTRAINT unique_local_account UNIQUE (local_agent_id, login),
    CONSTRAINT local_accounts_agent_fkey FOREIGN KEY (local_agent_id) 
        REFERENCES local_agents (id) ON UPDATE RESTRICT ON DELETE CASCADE
)
```

### Table ``remote_accounts``

```sqlite
CREATE TABLE remote_accounts (
    id              BIGINT       NOT NULL AUTOINCREMENT,
    remote_agent_id BIGINT       NOT NULL,
    login           VARCHAR(100) NOT NULL,
    password        TEXT         NOT NULL DEFAULT '',
    
    CONSTRAINT remote_accounts_pkey PRIMARY KEY (id),
    CONSTRAINT unique_remote_account UNIQUE(remote_agent_id, login),
    CONSTRAINT remote_accounts_agent_fkey FOREIGN KEY (remote_agent_id)
        REFERENCES remote_agents (id) ON UPDATE RESTRICT ON DELETE CASCADE
)
```

### Table ``rules``

```sqlite
CREATE TABLE rules (
    id                    BIGINT       NOT NULL AUTOINCREMENT,
    name                  VARCHAR(100) NOT NULL,
    comment               TEXT         NOT NULL DEFAULT '',
    is_send               BOOLEAN      NOT NULL,
    path                  VARCHAR(100) NOT NULL,
    local_dir             TEXT         NOT NULL DEFAULT '',
    remote_dir            TEXT         NOT NULL DEFAULT '',
    tmp_local_receive_dir TEXT         NOT NULL DEFAULT '',
    
    CONSTRAINT rules_pkey PRIMARY KEY (id),
    CONSTRAINT unique_rule_name UNIQUE (is_send, name),
    CONSTRAINT unique_rule_path UNIQUE (is_send, path)
)
```

### Table ``tasks``

```sqlite
CREATE TABLE tasks (
    rule_id BIGINT      NOT NULL,
    chain   VARCHAR(10) NOT NULL,
    rank    TINYINT     NOT NULL,
    type    VARCHAR(50) NOT NULL,
    args    TEXT        NOT NULL DEFAULT '{}',
      
    CONSTRAINT unique_task_nb UNIQUE (rule_id, chain, rank),
    CONSTRAINT tasks_rule_id_fkey FOREIGN KEY (rule_id) 
        REFERENCES rules (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT task_rank_check    CHECK (rank >= 0),
    CONSTRAINT task_chain_check   CHECK (chain = 'PRE' OR
        chain = 'POST' OR chain = 'ERROR')
)
```

### Table ``rule_access``

```sqlite
CREATE TABLE rule_access (
    rule_id           BIGINT NOT NULL,
    local_agent_id    BIGINT,
    remote_agent_id   BIGINT,
    local_account_id  BIGINT,
    remote_account_id BIGINT,
    
    CONSTRAINT unique_access_loc_agent   UNIQUE (rule_id, local_agent_id),
    CONSTRAINT unique_access_rem_agent   UNIQUE (rule_id, remote_agent_id),
    CONSTRAINT unique_access_loc_account UNIQUE (rule_id, local_account_id),
    CONSTRAINT unique_access_rem_account UNIQUE (rule_id, remote_account_id),
    CONSTRAINT access_local_account_id_fkey FOREIGN KEY (`local_account_id`)
        REFERENCES local_accounts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT access_local_agent_id_fkey FOREIGN KEY (local_agent_id)
        REFERENCES local_agents (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT access_remote_account_id_fkey FOREIGN KEY (remote_account_id)
        REFERENCES remote_accounts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT access_remote_agent_id_fkey FOREIGN KEY (remote_agent_id)
        REFERENCES remote_agents (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT access_rule_id_fkey FOREIGN KEY (rule_id)
        REFERENCES rules (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT rule_access_target_check CHECK (
        CASE WHEN local_agent_id    IS NOT NULL THEN 1 ELSE 0 END + 
        CASE WHEN remote_agent_id   IS NOT NULL THEN 1 ELSE 0 END + 
        CASE WHEN local_account_id  IS NOT NULL THEN 1 ELSE 0 END + 
        CASE WHEN remote_account_id IS NOT NULL THEN 1 ELSE 0 END = 1)
)
```

### Table ``crypto_credentials``

```sqlite
CREATE TABLE crypto_credentials (
    id                BIGINT       NOT NULL AUTOINCREMENT,
    name              VARCHAR(100) NOT NULL,
    local_agent_id    BIGINT,
    remote_agent_id   BIGINT,
    local_account_id  BIGINT,
    remote_account_id BIGINT,
    private_key       TEXT         NOT NULL DEFAULT '',
    certificate       TEXT         NOT NULL DEFAULT '',
    ssh_public_key    TEXT         NOT NULL DEFAULT '',
    
    
    CONSTRAINT crypto_credentials_pkey PRIMARY KEY (id),
    CONSTRAINT unique_crypto_loc_agent   UNIQUE (local_agent_id, name),
    CONSTRAINT unique_crypto_rem_agent   UNIQUE (remote_agent_id, name),
    CONSTRAINT unique_crypto_loc_account UNIQUE (local_account_id, name),
    CONSTRAINT unique_crypto_rem_account UNIQUE (remote_account_id, name),
    CONSTRAINT crypto_local_account_fkey FOREIGN KEY (local_account_id)
        REFERENCES local_accounts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT crypto_local_agent_fkey FOREIGN KEY (local_agent_id)
        REFERENCES local_agents (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT crypto_remote_account_fkey FOREIGN KEY (remote_account_id)
        REFERENCES remote_accounts (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT crypto_remote_agent_fkey FOREIGN KEY (remote_agent_id)
        REFERENCES remote_agents (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT crypto_check_owner CHECK (
        CASE WHEN local_agent_id    IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN remote_agent_id   IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN local_account_id  IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN remote_account_id IS NOT NULL THEN 1 ELSE 0 END = 1)
)
```

### Table ``transfers``

```sqlite
CREATE TABLE transfers (
    id                 BIGINT       NOT NULL AUTOINCREMENT,
    remote_transfer_id VARCHAR(100) NOT NULL,
    rule_id            BIGINT       NOT NULL,
    local_account_id   BIGINT,
    remote_account_id  BIGINT,
    src_filename       TEXT         NOT NULL DEFAULT '',
    dest_filename      TEXT         NOT NULL DEFAULT '',
    local_path         TEXT         NOT NULL DEFAULT '',
    remote_path        TEXT         NOT NULL DEFAULT '',
    start              DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    step               VARCHAR(50)  NOT NULL DEFAULT 'StepNone',
    status             VARCHAR(50)  NOT NULL DEFAULT 'PLANNED',
    owner              VARCHAR(100) NOT NULL,
    progress           BIGINT       NOT NULL DEFAULT 0,
    task_number        TINYINT      NOT NULL DEFAULT 0,
    error_code         VARCHAR(50)  NOT NULL DEFAULT 'TeOk',
    error_details      TEXT         NOT NULL DEFAULT '',
    filesize           BIGINT       NOT NULL DEFAULT -1,
    
    CONSTRAINT transfers_pkey PRIMARY KEY (id),
    CONSTRAINT unique_transfer_local  UNIQUE (remote_transfer_id, local_account_id),
    CONSTRAINT unique_transfer_remote UNIQUE (remote_transfer_id, remote_account_id),
    CONSTRAINT transfers_local_account_id_fkey FOREIGN KEY (local_account_id)
        REFERENCES local_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT transfers_remote_account_id_fkey FOREIGN KEY (remote_account_id)
        REFERENCES remote_accounts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT transfers_rule_id_fkey FOREIGN KEY (rule_id)
        REFERENCES rules (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT transfer_check_requester CHECK (
        CASE WHEN local_account_id  IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN remote_account_id IS NOT NULL THEN 1 ELSE 0 END = 1)
)
```

### Table ``transfer_history``

```sqlite
CREATE TABLE transfer_history (
    id                 BIGINT       NOT NULL,
    owner              VARCHAR(100) NOT NULL,
    remote_transfer_id VARCHAR(100) NOT NULL,
    is_server          BOOLEAN      NOT NULL,
    is_send            BOOLEAN      NOT NULL,
    account            VARCHAR(100) NOT NULL,
    agent              VARCHAR(100) NOT NULL,
    protocol           VARCHAR(50)  NOT NULL,
    src_filename       TEXT         NOT NULL DEFAULT '',
    dest_filename      TEXT         NOT NULL DEFAULT '',
    remote_path        TEXT         NOT NULL DEFAULT '',
    local_path         TEXT         NOT NULL DEFAULT '',
    rule               VARCHAR(100) NOT NULL,
    start              DATETIME     NOT NULL,
    stop               DATETIME,
    status             VARCHAR(50)  NOT NULL,
    error_code         VARCHAR(50)  NOT NULL DEFAULT 'TeOk',
    error_details      TEXT         NOT NULL DEFAULT '',
    step               VARCHAR(50)  NOT NULL,
    progress           BIGINT       NOT NULL DEFAULT 0,
    task_number        TINYINT      NOT NULL DEFAULT 0,
      
    CONSTRAINT transfer_history_pkey PRIMARY KEY (id),
    CONSTRAINT unique_history UNIQUE (remote_transfer_id, is_server, account, agent)
)
```

### Table ``transfer_info``

```sqlite
CREATE TABLE transfer_info (
    transfer_id BIGINT,
    history_id  BIGINT,
    name        VARCHAR(100) NOT NULL,
    value       TEXT         NOT NULL DEFAULT 'null',
  
    CONSTRAINT unique_transfer_info UNIQUE (transfer_id,name),
    CONSTRAINT unique_history_info  UNIQUE (history_id,name),
    CONSTRAINT info_history_fkey FOREIGN KEY (history_id)
        REFERENCES transfer_history (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT info_transfer_fkey FOREIGN KEY (transfer_id)
        REFERENCES transfers (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT transfer_info_check_owner CHECK (
        CASE WHEN transfer_id IS NOT NULL THEN 1 ELSE 0 END + 
        CASE WHEN history_id  IS NOT NULL THEN 1 ELSE 0 END = 1)
)
```